import argparse
import json
import re
import requests
import sys

def log(s):
  sys.stderr.write(s)

class Chunk:
  BlankLine, CommentLine, OneLineEntry, Unknown = range(4)

def getPendingRemovals():
  log("Fetching pending removal...\n")
  return requests.get("https://hstspreload.org/api/v2/pending-removal").json()

def getRawText(preloadListPath):
  log("Fetching preload list from Chromium source...\n")
  with open(preloadListPath, "r") as f:
      s = f.read()
  return s

def getPendingScan(pendingDataFilePath):
  log("Fetching pending list from provided path...\n")
  log("  %s\n" % pendingDataFilePath)
  with open(pendingDataFilePath, "r") as f:
      return json.load(f)

def domainsToPreload(pendingData, domainsToReject):
  numSkipping = 0
  numPreloading = 0
  for result in pendingData:
    if len(result["issues"]["errors"]) == 0:
      numPreloading += 1
      yield result["domain"]
    else:
      errors = list(error["code"] for error in result["issues"]["errors"])
      domainsToReject += [
        {"domain": result["domain"], "errors": errors}
      ]
      numSkipping += 1
  log("Pending entries preloaded: %d\n" % numPreloading)
  log("Pending entries rejected: %d\n" % numSkipping)

def chunks(rawText):
  log("Chunking...\n")
  lines = iter(rawText.splitlines())
  while True:
    try:
      chunk = next(lines)
      if chunk == "":
        yield chunk, Chunk.BlankLine
        continue
      elif re.match(r'^ *//.*', chunk):
        yield chunk, Chunk.CommentLine
        continue
      elif re.match(r'^    \{.*\},', chunk):
        yield chunk, Chunk.OneLineEntry
      else:
        yield chunk, Chunk.Unknown
    except StopIteration:
      break

def update(pendingRemovals, pendingAdditions, entryStrings):
  log("Removing and adding entries...\n")
  removedCount = 0
  for l, c in entryStrings:
    if c == Chunk.OneLineEntry:
      parsed = json.loads("[%s{}]" % l)[0]
      domain = parsed["name"]
      if domain in pendingRemovals:
        removedCount += 1
        pendingRemovals.remove(domain)
      else:
        yield l
    elif l == "    // END OF 1-YEAR BULK HSTS ENTRIES":
      for domain in sorted(pendingAdditions):
        yield '    { "name": "%s", "policy": "bulk-1-year", "mode": "force-https", "include_subdomains": true },' % domain
      yield l
    else:
      yield l
  log("Removed: %s\n" % removedCount)

def write(file, output):
  log("Writing to %s...\n" % file)
  with open(file, 'w') as file:
    file.write(output)
    file.close()

def getArgs():
  parser = argparse.ArgumentParser(description='Roll the HSTS preload list (experimental).')
  parser.add_argument('preload_list_path', type=str)
  parser.add_argument('pending_scan_path', type=str)
  parser.add_argument('rejected_domains_path', type=str)
  parser.add_argument('--skip_removals', action='store_true')
  return parser.parse_args()

def parseJsonWithComments(rawText):
  s = ""
  for l, c in chunks(rawText):
    if c == Chunk.CommentLine:
      continue
    else:
      s += l + "\n"
  return json.loads(s)

def checkForDupes(parsedList):
  log("Checking for duplicates...\n")
  seen = set()
  dupes = set()
  for entry in parsedList["entries"]:
    name = entry["name"]
    if name in seen:
      dupes.add(name)
    else:
      seen.add(name)
  return dupes

def main():
  args = getArgs()

  rawText = getRawText(args.preload_list_path)
  pendingRemovals = []
  if not args.skip_removals:
    pendingRemovals = getPendingRemovals()
  domainsToReject = []
  pendingAdditions = domainsToPreload(getPendingScan(args.pending_scan_path), domainsToReject)
  updated = update(pendingRemovals, pendingAdditions, chunks(rawText))
  updatedText = "\n".join(updated) + "\n"

  dupes = checkForDupes(parseJsonWithComments(updatedText))

  write(args.preload_list_path, updatedText)
  write(args.rejected_domains_path, json.dumps(domainsToReject, indent=2) + "\n")

  if dupes:
    print("\nWARNING\nDuplicate entries:")
    for dupe in dupes:
      print("- %s" % dupe)
    print("\nYou'll need to manually deduplicate entries before commiting them to Chromium.")
    print("\nNote: if there are a lot of duplicate entries, you may have accidentally run this script twice. Reset your checkout and try again.")
  else:
    print("\nSUCCESS\n")


if __name__ == "__main__":
    main()
