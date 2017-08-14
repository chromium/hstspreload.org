import base64
import json
import re
import requests
import sys

def log(s):
  sys.stderr.write(s)

class State:
  BeforeBulkEntries, DuringBulkEntries, AfterBulkEntries = range(3)

def getRawText():
  log("Fetching preload list from Chromium source...\n")
  return base64.b64decode(requests.get("https://chromium.googlesource.com/chromium/src/+/master/net/http/transport_security_state_static.json?format=TEXT").text)

def extractBulkEntries(rawText):
  log("Extracting bulk entries...\n")
  state = State.BeforeBulkEntries
  bulkEntryString = "[\n"
  for line in rawText.splitlines():
    if state == State.BeforeBulkEntries:
      if "START OF BULK ENTRIES" in line:
        state = State.DuringBulkEntries
    elif state == State.DuringBulkEntries:
      if "END OF BULK ENTRIES" in line:
        state = State.AfterBulkEntries
      else:
        bulkEntryString += line + "\n"
    elif state == State.AfterBulkEntries:
      if "BULK ENTRIES" in line:
        raise Exception("Preload list contains unexpected bulk entry markers.")

  # Add an empty object for the last entry to go after the trailing comma.
  bulkEntryString += "{}]"

  entries = json.loads(bulkEntryString)
  # Remove empty object at the end.
  del entries[-1]
  log("Found %d bulk entries.\n" % len(entries))
  return entries

def sanityCheck(domainList):
  log("Sanity checking domains...\n")
  for domain in domainList:
    log("\033[K\rChecking: %s" % domain)
    if not re.match(r'^[a-z0-9-\.]+$', domain):
      raise Exception("Incorrectly formatted domain: %s" % domain)
    if domain in ["google.com", "gmail.com", "hstspreload.org"]:
      raise Exception("Unexpected domain in list")
  log("\n")

def formatForGo(domainList):
  obj = {}
  for domain in domainList:
    obj[domain] = True
  return obj

def write(bulkDomains):
  log("Writing...\n")
  with open(sys.argv[1], 'w') as file:
    json.dump(formatForGo(bulkDomains), file)

def main():
  rawText = getRawText()
  bulkEntries = extractBulkEntries(rawText)
  bulkDomains = [entry["name"] for entry in bulkEntries]
  sanityCheck(bulkDomains)
  write(bulkDomains)
  log("\033[92mStatic bulk domain data update done!\x1b[0m\n")

if __name__ == "__main__":
    main()