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

def getRawText():
  log("Fetching preload list from Chromium source...\n")
  with open("/Users/lgarron/alt/src/net/http/transport_security_state_static.json", "r") as f:
      s = f.read()
  return s

def chunks(rawText):
  log("Iterating...\n")
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

def removePendingRemovals(pendingRemovals, entryStrings):
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
    else:
      yield l
  log("Removed: %s" % removedCount)

# def write(bulkDomains):
#   log("Writing...\n")
#   with open(sys.argv[1], 'w') as file:
#     json.dump(formatForGo(bulkDomains), file)

def main():
  rawText = getRawText()
  pendingRemovals = getPendingRemovals()
  removalsDone = removePendingRemovals(pendingRemovals, chunks(rawText))
  for l in removalsDone:
    print l
  log(str(pendingRemovals))

if __name__ == "__main__":
    main()