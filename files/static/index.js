"use strict";

var $ = document.querySelector.bind(document);

function PreloadSubmission() {
  this.setDomainFromURLParam();

  $("#domain-form").addEventListener("submit", function(ev) {
    this.onDomainChanged();
    ev.preventDefault();
  }.bind(this));

  $("#checkbox-owner").addEventListener("change", this.onCheckBoxChanged.bind(this));
  $("#checkbox-subdomains").addEventListener("change", this.onCheckBoxChanged.bind(this));
  $("#submit-form").addEventListener("submit", function(ev) {
    this.onSubmitForm();
    ev.preventDefault();
  }.bind(this));
}

PreloadSubmission.prototype = {
  // TODO: look at response codes.
  status: function(domain) {
    return new Promise(function(resolve, reject) {
      var oReq = new XMLHttpRequest();
      oReq.addEventListener("load", (function(ev) {
        resolve(JSON.parse(oReq.response));
      }).bind(this));
      oReq.open("GET", "/status/" + domain);
      oReq.send();
    });
  },

  checkdomain: function(domain) {
    console.log("checkdomain:" + domain);
    return new Promise(function(resolve, reject) {
      var oReq = new XMLHttpRequest();
      oReq.addEventListener("load", (function(ev) {
        resolve(JSON.parse(oReq.response));
      }).bind(this));
      oReq.open("GET", "/checkdomain/" + domain);
      oReq.send();
    });
  },

  submit: function(domain) {
    console.log("checkdomain:" + domain);
    return new Promise(function(resolve, reject) {
      var oReq = new XMLHttpRequest();
      oReq.addEventListener("load", (function(ev) {
        resolve(JSON.parse(oReq.response));
      }).bind(this));
      oReq.open("GET", "/submit/" + domain);
      oReq.send();
    });
  },

  isValidDomainFormat: function(domain) {
    return domain.includes(".") &&
          !domain.endsWith(".") &&
          !domain.startsWith(".") &&
          !domain.includes("..");
  },

  // hasLikelyETLD1: function(domain) {
  //   // Submitting .com domains is common; don't use the fast path for .co
  //   // Same for .net => .ne
  //   if (domain.endsWith(".co") || domain.endsWith(".ne")) {
  //     return true;
  //   }

  //   var publicsuffix = tldjs.getPublicSuffix(domain)
  //   return publicsuffix !== null && publicsuffix !== domain;
  // },

  clearTheme: function(theme) {
    document.body.classList.remove("theme-green", "theme-yellow", "theme-red");
  },

  setTheme: function(theme) {
    this.clearTheme();
    document.body.classList.add(theme)
  },

  currentDomain: function() {
    var domain = $("#domain").value;
    // Check for pasted URLs beginning with http:// or https://
    if (domain.startsWith("http://")) {
      domain = domain.slice("http://".length)
    }
    if (domain.startsWith("https://")) {
      domain = domain.slice("https://".length)
    }
    return domain;
  },

  clearResults: function() {
    $("#result").classList.add("hidden");
  },

  showResults: function(domain, checkDomainIssues, domainStatus) {
    if (checkDomainIssues.errors.length === 0) {
      if (checkDomainIssues.warnings.length === 0) {
        $("#summary").textContent = "" + domain + " satisfies all requirements for preloading!";
        this.setTheme("theme-green");
      } else {
        $("#summary").textContent = "" + domain + " has " + (checkDomainIssues.warnings.length == 1 ? "a warning" : "warnings") + ", but satisfies all requirements for preloading.";
        this.setTheme("theme-yellow");
      }
    } else {
        $("#summary").textContent = "" + domain + " has errors and cannot be preloaded.";
        this.setTheme("theme-red");
    }

    switch (domainStatus.status) {
    case "unknown":
      if (checkDomainIssues.errors.length === 0) {
        $("#status").textContent = "" + domain + " is not yet preloaded, and may be submitted to the preload list."
      } else {
        $("#status").textContent = "" + domain + " is not preloaded."
      }
      break;
    case "pending":
      $("#status").textContent = "" + domain + " is pending submission to the preload list."
      break;
    case "preloaded":
      $("#status").textContent = "" + domain + " is currently preloaded."
      break;
    case "rejected":
      // TODO: do something useful if domainStatus.message is not present.
      if (domainStatus.message) {
        $("#status").textContent = "" + domain + " was rejected from the preload list for the following reason: " + domainStatus.message;
      } else {
        $("#status").textContent = "" + domain + " was rejected from the preload list.";
      }
      if (checkDomainIssues.errors.length === 0) {
        $("#status").textContent += " It may be submitted again."
      }
      break;
    case "removed":
      if (checkDomainIssues.warnings.length === 0) {
        $("#status").textContent = "" + domain + " was previously on the preload list, but has been removed. It may be submitted again.";
      } else {
        $("#status").textContent = "" + domain + " was previously on the preload list, but has been removed.";
      }
      break;
    default:
      $("#status").textContent = "Cannot determine preload status.";
    }

    function createIssueElement(issue, bullet, className) {
      var el = document.createElement("div");
      el.classList.add(className);
      // summary.title = issue.code;

      var bulletSpan = document.createElement("span");
      bulletSpan.textContent = bullet + " ";
      el.appendChild(bulletSpan);

      var summary = document.createElement("span")
      summary.classList.add("summary");
      summary.textContent = issue.summary;
      el.appendChild(summary);

      var br = document.createElement("br")
      el.appendChild(br);

      var message = document.createElement("span")
      message.textContent = issue.message;
      el.appendChild(message);

      return el;
    }

    $("#errors").textContent = "";
    $("#warnings").textContent = "";
    for (var i in checkDomainIssues.errors) {
      var error = checkDomainIssues.errors[i];
      var el = createIssueElement(error, "❌", "error");
      $("#errors").appendChild(el);
    }
    for (var j in checkDomainIssues.warnings) {
      var warning = checkDomainIssues.warnings[j];
      var el = createIssueElement(warning, "⚠️", "warning");
      $("#warnings").appendChild(el);
    }

    $("#result").classList.remove("hidden");
  },

  hideSubmission: function() {
    $("#submit-form").classList.add("hidden");
  },

  showSubmission: function() {
    $("#checkbox-owner").checked = false;
    $("#checkbox-subdomains").checked = false;
    $("#submit").disabled = true;

    $("#submit-form").classList.remove("hidden");
  },

  onSubmitForm: function() {
    this.submit(this.domainToSubmit).then(function(issues) {
      console.log("submit:", issues);
      $("#submit-result").classList.remove("hidden");
      if (issues.errors.length == 0) {
        $("#submit-result").textContent = "Submitted successfully!";
        // TODO: Now try SSL Labs!
      } else {
        $("#submit-result").textContent = "There are errors. Please submit your site again.";
      }
    });
  },

  showWaiting: function(domain) {
    $("#checking").textContent = "Checking  " + domain;

    $("#result-waiting").classList.remove("hidden");
  },

  hideWaiting: function() {
    $("#result-waiting").classList.add("hidden");
  },

  updateURLHash: function(domain) {
    console.log(domain);
    if (domain) {
      history.replaceState({}, document.title, ".?domain=" + domain);
    } else {
      history.replaceState({}, document.title, ".");
    }
  },

  onDomainChanged: function() {
    var domain = this.currentDomain();
    this.showWaiting(domain);
    this.clearResults();
    this.hideSubmission();
    this.updateURLHash(domain);
    this.clearTheme()

    if (!domain) {
      $("#result-waiting").classList.add("hidden");
      return;
    }

    this.currentResultsDomain = "";

    Promise.all([
      this.checkdomain(domain),
      this.status(domain)
    ]).then(function(values) {
      this.handleResults(domain, values[0], values[1]);
    }.bind(this), function() {
        // TODO: handle failure better.
        $("#result").classList.remove("hidden");
        $("#result-waiting").classList.add("hidden");
    });
  },

  onCheckBoxChanged: function() {
    $("#submit").disabled = !($("#checkbox-owner").checked && $("#checkbox-subdomains").checked);
  },

  handleResults: function(domain, checkDomainIssues, domainStatus) {
    console.log("handleResults:", checkDomainIssues, domainStatus);

    if (domain !== this.currentDomain()) {
      return;
    }
    if (this.currentResultsDomain === domain) {
      return;
    } else {
      this.currentResultsDomain = domain;
    }

    if (domain !== this.currentDomain()) {
      console.log("Outdated result.");
      return;
    }

    this.showResults(domain, checkDomainIssues, domainStatus);
    this.hideWaiting();

    if (checkDomainIssues.errors.length === 0 && ["unknown", "rejected", "removed"].indexOf(domainStatus.status) != -1) {
      this.domainToSubmit = domain;
      this.showSubmission();
    }
  },

  domainFromURLParam: function() {
    var match = window.location.search.match(/^\?domain=([^?&]+)$/);
    return match ? decodeURIComponent(match[1]) : null;
  },

  setDomainFromURLParam: function() {
    var domain = this.domainFromURLParam();
    if (domain) {
      console.log("From URL parameter:", domain);
      $("#domain").value = domain;
      this.onDomainChanged();
    }
  }

  // showResult: function(domain, status) {
  //   console.log(status);   

  //   this.showElement("resultError");
  //   $("#resultSummary").textContent = domain + "hi there"

  //   // ✅

  //   var errorsElem = $("#resultError").querySelector(".errors")
  //   for (var i in status.errors) {
  //     console.log("aaa")
  //     var li = document.createElement("li");
  //     li.textContent = "❌ " + status.errors[i]
  //     errorsElem.appendChild(li);
  //   }

  //   var warningsElem = $("#resultError").querySelector(".warnings")
  //   for (var i in status.warnings) {
  //     var li = document.createElement("li");
  //     li.textContent = "⚠️ " + status.warnings[i]
  //     warningsElem.appendChild(li);
  //   }
  // },

  // showElement: function(id) {
  //   var ids = ["loading", "domainForm", "resultError"]
  //   for (var i in ids) {
  //     document.getElementById(ids[i]).style.display = (ids[i] == id) ? "block" : "block";
  //   }
  // }
}

window.addEventListener("load", function() {
  new PreloadSubmission();
});
