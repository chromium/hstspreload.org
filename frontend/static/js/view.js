'use strict';

/* global $:false, extractDomain:false */

/*
  * verb: "add" or "remove"
 */
var PreloadView = function(controller, submitDomain, urlParam) {
  this._controller = controller;
  this._urlParam = urlParam;

  var submitDomainHandler = function(ev) {
    ev.preventDefault();
    submitDomain();
  };

  $('#domain-form').addEventListener('input', this._DomainInput.bind(this));

  $('#submit-form').addEventListener('submit', submitDomainHandler);

  if (this._controller.formHasCheckboxes()) {
    $('#checkbox-subdomains')
      .addEventListener('change', this._checkboxChangedHandler.bind(this));
  }
};

PreloadView.prototype = {
  _removeHash: function() {
    history.replaceState(
      {}, document.title, window.location.pathname + window.location.search);
  },

  _checkboxChangedHandler: function() {
    $('#submit').disabled = !$('#checkbox-subdomains').checked;
  },

  clearTheme: function() {
    document.body.classList.remove('theme-green', 'theme-yellow', 'theme-red');
  },

  setTheme: function(theme) {
    this.clearTheme();
    document.body.classList.add(theme);
  },

  _DomainInput: function() {
    this._removeHash();
    this.clearDomainSpecificElements();
  },

  clearDomainSpecificElements: function() {
    this._hideResults();
    this._clearIssues();
    this._hideSubmission();
    this.setSummary('');
    this.setStatus('');
    this.clearTheme();
    this.hideWaiting();
  },

  clickCheck() {
    $('#check').click();
  },

  setDomain(domain) {
    $('#domain').value = domain;
  },

  currentDomain: function() {
    var domain = $('#domain').value.trim();
    return extractDomain(domain);
  },

  showWaiting: function(domain) {
    $('#checking').textContent = 'Checking  ' + domain;
    $('#result-waiting').classList.remove('hidden');
  },

  hideWaiting: function() {
    $('#result-waiting').classList.add('hidden');
  },

  showResults: function() {
    $('#result').classList.remove('hidden');
  },

  _hideResults: function() {
    $('#result').classList.add('hidden');
  },

  showSubmission: function(domain) {
    if (this._controller.formHasCheckboxes()) {
      $('#checkbox-subdomains').checked = false;
      $('#submit').disabled = true;
    } else {
      $('#submit').disabled = false;
    }
    document.getElementById('submit').value = this._controller.submitButtonString(domain);

    var domainTexts = document.getElementsByClassName('domain-text');
    for (var i = 0; i < domainTexts.length; i++) {
      domainTexts[i].textContent = domain;
    }

    $('#submit-form').classList.remove('hidden');
  },

  _hideSubmission: function() {
    $('#submit-success').hide();
    $('#submit-failure').hide();
    if ($('#ssl-labs-link')) {
      $('#ssl-labs-link').href = 'https://www.ssllabs.com/ssltest/analyze.html';
    }
    $('#submit-form').hide();
  },

  clearSummary: function() {
    this.setSummary('');
  },

  setSummary: function(summaryMessage) {
    $('#summary').textContent = summaryMessage;
  },

  clearStatus: function() {
    this.setStatus('');
  },

  setStatus: function(statusMessage) {
    $('#status').textContent = statusMessage;
  },

  // TODO: remove
  _clearIssues: function() {
    $('#issues-wrapper').textContent = '';
  },

  showIssues: function(issues) {
    $('#issues-wrapper').appendChild(new IssuesBulletList(issues));
  }
};

/* returns Element */
var IssueBullet = function(issue, type, typeLabel) {
  if (['error', 'warning'].indexOf(type) === -1) {
    throw new Error('Unknown type of issue.');
  }

  var el = document.createElement('div');
  el.classList.add(type);

  el.createChild('img', 'bullet').src = '/static/img/' + type + '.svg';
  el.createChild('span', 'summary').textContent =
      typeLabel + ': ' + issue.summary;
  el.createChild('span', 'message').textContent = issue.message;
  return el;
};

/* returns Element */
var IssuesBulletList = function(issues) {
  var el = document.createElement('div');
  el.classList.add('issues');

  var errorsElem = el.createChild('div', 'errors');
  errorsElem.classList.add('issues-list');
  var warningsElem = el.createChild('div', 'warnings');
  warningsElem.classList.add('issues-list');

  for (var e of issues.errors) {
    errorsElem.appendChild(new IssueBullet(e, 'error', 'Error'));
  }
  for (var w of issues.warnings) {
    warningsElem.appendChild(new IssueBullet(w, 'warning', 'Warning'));
  }
  return el;
};
