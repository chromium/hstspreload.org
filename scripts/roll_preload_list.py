import argparse
import base64
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

def getRawText(preload_list_path):
  log("Fetching preload list from Chromium source...\n")
  with open(preload_list_path, "r") as f:
      s = f.read()
  return s

def getPendingScan(pendingDataFilePath):
  log("Fetching pending list from provided path...\n")
  with open(pendingDataFilePath, "r") as f:
      return json.load(f)

def domainsToPreload(pendingData):
  numSkipping = 0
  numPreloading = 0
  for result in pendingData:
    if len(result["issues"]["errors"]) == 0:
      numPreloading += 1
      yield result["domain"]
    else:
      numSkipping += 1
  log("Pending entries preloaded: %d\n" % numPreloading)
  log("Pending entries skipped: %d\n" % numSkipping)

def chunks(rawText):
  log("Chunking...\n")
  lines = iter(rawText.splitlines())
  while True:
    try:
      chunk = next(lines)
      if chunk == "":
        yield chunk, Chunk.BlankLine
        continue
      elif re.match(r'^ +//.*', chunk):
        yield chunk, Chunk.CommentLine
        continue
      elif re.match(r'^    \{.*\},', chunk):
        yield chunk, Chunk.OneLineEntry
      else:
        yield chunk, Chunk.Unknown
    except StopIteration:
      break

def update(pendingRemovals, pendingAdditions, entryStrings):
  log("Updating...\n")
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
    elif l == "    // END OF BULK ENTRIES":
      for domain in pendingAdditions:
        yield '    { "name": "%s", "include_subdomains": true, "mode": "force-https" },' % domain
      yield l
    else:
      yield l
  log("Removed: %s\n" % removedCount)

def write(preload_list_path, output):
  log("Overwriting preload list source...\n")
  with open(preload_list_path, 'w') as file:
    file.write(output)

def getArgs():
  parser = argparse.ArgumentParser(description='Roll the HSTS preload list (experimental).')
  parser.add_argument('preload_list_path', type=str)
  parser.add_argument('pending_scan_path', type=str)
  return parser.parse_args()

def main():
  args = getArgs()

  rawText = getRawText(args.preload_list_path)
  pendingRemovals = getPendingRemovals()
  pendingAdditions = domainsToPreload(getPendingScan(args.pending_scan_path))
  removalsDone = update(pendingRemovals, pendingAdditions, chunks(rawText))

  output = ""
  for l in removalsDone:
    output += l + "\n"
  write(args.preload_list_path, output)

if __name__ == "__main__":
    main()