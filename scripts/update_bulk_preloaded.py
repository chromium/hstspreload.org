"""Update the bulk HSTS preload list."""

from __future__ import print_function
import base64
import json
import re
import requests
import sys


def log(s):
  sys.stderr.write(s)


class State(object):
  BEFORE_LEGACY_18WEEK_BULK_ENTRIES, \
  DURING_LEGACY_18WEEK_BULK_ENTRIES, \
  AFTER_LEGACY_18WEEK_BULK_ENTRIES, \
  DURING_18WEEK_BULK_ENTRIES, \
  AFTER_18WEEK_BULK_ENTRIES, \
  DURING_1YEAR_BULK_ENTRIES, \
  AFTER_1YEAR_BULK_ENTRIES, \
  DURING_1YEAR_BULK_SUBDOMAIN_ENTRIES, \
  AFTER_1YEAR_BULK_SUBDOMAIN_ENTRIES = list(range(9))


def get_raw_text():
  log("Fetching preload list from Chromium source...\n")
  return base64.b64decode(
      requests.get(
          "https://chromium.googlesource.com/chromium/src/+/main/net/http/"
          "transport_security_state_static.json?format=TEXT"
      ).text).decode("UTF-8")


def extract_bulk_entries(raw_text):
  log("Extracting bulk entries...\n")
  state = State.BEFORE_LEGACY_18WEEK_BULK_ENTRIES
  bulk_entry_string = "[\n"
  for line in raw_text.splitlines():
    if state == State.BEFORE_LEGACY_18WEEK_BULK_ENTRIES:
      if "START OF LEGACY 18-WEEK BULK HSTS ENTRIES" in line:
        state = State.DURING_LEGACY_18WEEK_BULK_ENTRIES
    elif state == State.DURING_LEGACY_18WEEK_BULK_ENTRIES:
      if "END OF LEGACY 18-WEEK BULK HSTS ENTRIES" in line:
        state = State.AFTER_LEGACY_18WEEK_BULK_ENTRIES
      else:
        bulk_entry_string += line + "\n"
    if state == State.AFTER_LEGACY_18WEEK_BULK_ENTRIES:
      if "START OF 18-WEEK BULK HSTS ENTRIES" in line:
        state = State.DURING_18WEEK_BULK_ENTRIES
    elif state == State.DURING_18WEEK_BULK_ENTRIES:
      if "END OF 18-WEEK BULK HSTS ENTRIES" in line:
        state = State.AFTER_18WEEK_BULK_ENTRIES
      else:
        bulk_entry_string += line + "\n"
    if state == State.AFTER_18WEEK_BULK_ENTRIES:
      if "START OF 1-YEAR BULK HSTS ENTRIES" in line:
        state = State.DURING_1YEAR_BULK_ENTRIES
    elif state == State.DURING_1YEAR_BULK_ENTRIES:
      if "END OF 1-YEAR BULK HSTS ENTRIES" in line:
        state = State.AFTER_1YEAR_BULK_ENTRIES
      else:
        bulk_entry_string += line + "\n"
    elif state == State.AFTER_1YEAR_BULK_ENTRIES:
      if "START OF 1-YEAR BULK SUBDOMAIN HSTS ENTRIES" in line:
        state = State.DURING_1YEAR_BULK_SUBDOMAIN_ENTRIES
    elif state == State.DURING_1YEAR_BULK_SUBDOMAIN_ENTRIES:
      if "END OF 1-YEAR BULK SUBDOMAIN HSTS ENTRIES" in line:
        state = State.AFTER_1YEAR_BULK_SUBDOMAIN_ENTRIES
      else:
        bulk_entry_string += line + "\n"
    elif state == State.AFTER_1YEAR_BULK_SUBDOMAIN_ENTRIES:
      if "BULK" in line:
        print(line)
        raise Exception("Preload list contains unexpected bulk entry markers.")
  if state != State.AFTER_1YEAR_BULK_SUBDOMAIN_ENTRIES:
    raise Exception(f"Unexpected end state: {state}")

  # Add an empty object for the last entry to go after the trailing comma.
  bulk_entry_string += "{}]"

  entries = json.loads(bulk_entry_string)
  # Remove empty object at the end.
  del entries[-1]
  log(f"Found {len(entries)} bulk entries.\n")
  return entries


def sanity_check(domain_list):
  log("Sanity checking domains...\n")
  for domain in domain_list:
    log(f"\033[K\rChecking: {domain}")
    if not re.match(r"^[a-z0-9-\.]+$", domain):
      raise Exception(f"Incorrectly formatted domain: {domain}")
    if domain in ["google.com", "gmail.com", "hstspreload.org"]:
      raise Exception("Unexpected domain in list")
  log("\n")


def format_for_go(domain_list):
  obj = {}
  for domain in domain_list:
    obj[domain] = True
  return obj


def write(bulk_domains):
  log("Writing...\n")
  with open(sys.argv[1], "w", encoding="utf-8") as file:
    json.dump(format_for_go(bulk_domains), file)


def main():
  raw_text = get_raw_text()
  bulk_entries = extract_bulk_entries(raw_text)
  bulk_domains = [entry["name"] for entry in bulk_entries]
  sanity_check(bulk_domains)
  write(bulk_domains)
  log("\033[92mStatic bulk domain data update done!\x1b[0m\n")


if __name__ == "__main__":
  main()
