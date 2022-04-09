"""Updates the HSTS preload list JSON file."""

from __future__ import print_function
import argparse
import json
import re
import requests
import sys


def log(s):
  sys.stderr.write(s)


class Chunk(object):
  BLANK_LINE, COMMENT_LINE, ONE_LINE_ENTRY, UNKNOWN = list(range(4))


def get_pending_removals():
  log("Fetching pending removal...\n")
  return requests.get("https://hstspreload.org/api/v2/pending-removal").json()


def get_raw_text(preload_list_path):
  log("Fetching preload list from Chromium source...\n")
  with open(preload_list_path, "r", encoding="utf-8") as f:
    s = f.read()
  return s


def get_pending_scan(pending_data_file_path):
  log("Fetching pending list from provided path...\n")
  log(f"  {pending_data_file_path}\n")
  with open(pending_data_file_path, "r", encoding="utf-8") as f:
    return json.load(f)


def domains_to_preload(pending_data, domains_to_reject):
  num_skipping = 0
  num_preloading = 0
  for result in pending_data:
    if len(result["issues"]["errors"]) == 0:
      num_preloading += 1
      yield result["domain"]
    else:
      errors = list(error["code"] for error in result["issues"]["errors"])
      domains_to_reject += [{"domain": result["domain"], "errors": errors}]
      num_skipping += 1
  log(f"Pending entries preloaded: {num_preloading}\n")
  log(f"Pending entries rejected: {num_skipping}\n")


def chunks(raw_text):
  log("Chunking...\n")
  lines = iter(raw_text.splitlines())
  while True:
    try:
      chunk = next(lines)
      if chunk == "":
        yield chunk, Chunk.BLANK_LINE
        continue
      elif re.match(r"^ *//.*", chunk):
        yield chunk, Chunk.COMMENT_LINE
        continue
      elif re.match(r"^    \{.*\},", chunk):
        yield chunk, Chunk.ONE_LINE_ENTRY
      else:
        yield chunk, Chunk.UNKNOWN
    except StopIteration:
      break


def update(pending_removals, pending_additions, entry_strings):
  log("Removing and adding entries...\n")
  removed_count = 0
  for l, c in entry_strings:
    if c == Chunk.ONE_LINE_ENTRY:
      # `l` will have a trailing comma -- remove it first, and then the line
      # can be directly parsed as JSON.
      parsed = json.loads(re.sub(r"},\w*$", r"}", l))
      domain = parsed["name"]
      if domain in pending_removals:
        removed_count += 1
        pending_removals.remove(domain)
      else:
        yield l
    elif l == "    // END OF 1-YEAR BULK HSTS ENTRIES":
      for domain in sorted(pending_additions):
        yield (f'    {{ "name": "{domain}", "policy": "bulk-1-year", '
               f'"mode": "force-https", "include_subdomains": true }},')
      yield l
    else:
      yield l
  log(f"Removed: {removed_count}\n")


def write(filename, output):
  log(f"Writing to {filename}...\n")
  with open(filename, "w", encoding="utf-8") as file:
    file.write(output)
    file.close()


def get_args():
  parser = argparse.ArgumentParser(
      description="Roll the HSTS preload list (experimental).")
  parser.add_argument("preload_list_path", type=str)
  parser.add_argument("pending_scan_path", type=str)
  parser.add_argument("rejected_domains_path", type=str)
  parser.add_argument("--skip_removals", action="store_true")
  return parser.parse_args()


def parse_json_with_comments(raw_text):
  s = ""
  for l, c in chunks(raw_text):
    if c == Chunk.COMMENT_LINE:
      continue
    else:
      s += l + "\n"
  return json.loads(s)


def check_for_dupes(parsed_list):
  log("Checking for duplicates...\n")
  seen = set()
  dupes = set()
  for entry in parsed_list["entries"]:
    name = entry["name"]
    if name in seen:
      dupes.add(name)
    else:
      seen.add(name)
  return dupes


def main():
  args = get_args()

  raw_text = get_raw_text(args.preload_list_path)
  pending_removals = []
  if not args.skip_removals:
    pending_removals = get_pending_removals()
  domains_to_reject = []
  pending_additions = \
      domains_to_preload(get_pending_scan(args.pending_scan_path),
                         domains_to_reject)
  updated = update(pending_removals, pending_additions, chunks(raw_text))
  updated_text = "\n".join(updated) + "\n"

  dupes = check_for_dupes(parse_json_with_comments(updated_text))

  write(args.preload_list_path, updated_text)
  write(args.rejected_domains_path,
        json.dumps(domains_to_reject, indent=2) + "\n")

  if dupes:
    print("\nWARNING\nDuplicate entries:")
    for dupe in dupes:
      print(f"- {dupe}")
    print(
        "\nYou'll need to manually deduplicate entries before commiting them "
        "to Chromium."
    )
    print(
        "\nNote: if there are a lot of duplicate entries, you may have "
        "accidentally run this script twice. Reset your checkout and try "
        "again."
    )
  else:
    print("\nSUCCESS\n")


if __name__ == "__main__":
  main()
