var requestedDomain = "";

function keypress(e) {
  var code = (e.keyCode ? e.keyCode : e.which);
  if (code != 13) {
    return;
  }

  var textInput = document.getElementById("textinput");
  textInput.disabled = true;
  var domain = textInput.value;

  clearOutput();

  if (domain.indexOf(":") != -1) {
    var parser = document.createElement('a');
    parser.href = domain;
    domain = parser.hostname;
  }

  if (domain === undefined || domain.length < 1) {
    showError("That doesn't look like a domain name.");
    return;
  }

  var xhr = new XMLHttpRequest();
  xhr.open("POST", "/submit/" + domain, true);
  xhr.onreadystatechange = handleReply;
  xhr.onerror = handleError;
  xhr.onTimeout = handleTimeout;
  xhr.timeout = 10000;
  requestedDomain = domain;
  xhr.send(null);
}

function handleReply(progress) {
  var xhr = progress.target;

  if (xhr.readyState != 4) {
    return;
  }

  if (xhr.status != 200) {
    showError("Request to server returned status " + xhr.status + ".");
    return;
  }

  var j = JSON.parse(xhr.responseText);
  if (j['Error'] != undefined) {
    showError(j['Error']);
    return;
  }

  var wholeDomainsMsg = "Only whole domains can be submitted because the interaction of cookies, HSTS and user behaviour is complex and we believe that only accepting whole domains is simple enough to have clear security semantics and usually the correct choice for sites.";

  if (j['Canon'] != undefined) {
    var textInput = document.getElementById("textinput");

    textInput.value = j['Canon'];
    textInput.focus();

    showMessage(wholeDomainsMsg + " (Based on public-suffix lists, " + requestedDomain + " is a subdomain of " + j['Canon'] + ".)");
    return;
  }

  if (j['Exception'] != undefined) {
    showMessage("This host failed manual review. The following message was set for it.");
    showMessage(j['Exception']);
    showMessage("You can remove this entry by clicking the button below. (This will allow you to resubmit if you so choose.)");

    var button = document.createElement('input');
    button.type = "submit";
    button.value = "Clear";
    document.getElementById("msg").appendChild(button);
    button.onclick = function(e) {
      document.getElementById("textinput").disabled = true;
      clearOutput();

      var xhr = new XMLHttpRequest();
      xhr.open("POST", "/clear/" + requestedDomain, true);
      xhr.onreadystatechange = handleClearReply;
      xhr.onerror = handleError;
      xhr.onTimeout = handleTimeout;
      xhr.timeout = 10000;
      xhr.send(null);
    }

    return;
  }

  if (j['IsPending']) {
    showMessage("That domain name has already been accepted and is pending review. Note that review can take several weeks.");
    return;
  }

  if (j['IsPreloaded']) {
    showMessage("That domain name is already preloaded! If you don't see it, note that changes follow the usual canary, dev, beta, stable progression and so can take several months to reach a stable release.");
    return;
  }

  if (j['NoHeader']) {
    showMessage("No HSTS header was found on that domain. See below about the requirements for preloading.");
    if (j['WasRedirect']) {
      showMessage("Note that the request resulted in a redirect. Ensure that the redirect itself has the HSTS header and not just the target page.");
    }
    return;
  }

  var bad = false;

  if (j['NoPreload']) {
    showMessage("The HSTS header on that site doesn't include a preload token. This is a non-standard token but it's used to authenticate the preload request.");
    bad = true;
  }

  if (j['MaxAge'] != undefined && j['MaxAge'] < 10886400) {
    showMessage("The max-age in the HSTS header is too short. It's currently " + j['MaxAge'] + " seconds, but it needs to be at least eighteen weeks (10886400 seconds).");
    bad = true;
  }

  if (j['NoSubdomains']) {
    showMessage("The includeSubdomains token is missing from the HSTS header. " + wholeDomainsMsg);
    bad = true;
  }

  if (bad) {
    showMessage("Fix the HSTS header and try again.");
    return;
  }

  if (j['Accepted']) {
    showMessage("Thank you! That domain has been queued for review. Review can take several weeks. You can check the status by entering the same domain again in the future.");

    var p = document.createElement('p');
    p.innerHTML = "Next, <a href=\"https://www.ssllabs.com/ssltest/analyze.html?d=" + requestedDomain + "\">check your HTTPS configuration</a> and fix any issues!";
    document.getElementById("msg").appendChild(p);

    document.getElementById("textinput").value = "";
    return;
  }

  showError("Bad reply from server");
}

function handleTimeout(xhr) {
  showError("Request timed out.");
}

function handleError(xhr) {
  showError("Request to server failed.");
}

function handleClearReply(progress) {
  var xhr = progress.target;

  if (xhr.readyState != 4) {
    return;
  }

  if (xhr.status != 200) {
    showError("Request to server returned status " + xhr.status + ".");
    return;
  }

  showMessage("Entry deleted");
}

function clearOutput() {
  var msgDiv = document.getElementById("msg");
  while (msgDiv.hasChildNodes()) {
    msgDiv.removeChild(msgDiv.lastChild);
  }
  document.getElementById("error").textContent = "";
}

function showError(msg) {
  document.getElementById("error").textContent = "Sorry! " + msg;
  document.getElementById("textinput").disabled = false;
}

function showMessage(msg) {
  var p = document.createElement('p');
  p.textContent = msg;
  document.getElementById("msg").appendChild(p);
  document.getElementById("textinput").disabled = false;
}
