'use strict';

var PreloadView = function(submitDomain, urlParam) {
  this._urlParam = urlParam;

  var submitDomainHandler = function(ev) {
    ev.preventDefault();
    submitDomain();
  };

  $('#domain-form').addEventListener('input', this._DomainInput.bind(this));

  $('#submit-form').addEventListener('submit', submitDomainHandler);

  $('#checkbox-owner')
      .addEventListener('change', this._checkboxChangedHandler.bind(this));
  $('#checkbox-subdomains')
      .addEventListener('change', this._checkboxChangedHandler.bind(this));

  if (location.hash === '') {
    $('#domain').focus()
  } else {
    this._highlightHash();
  }
  window.addEventListener('hashchange', this._highlightHash);
};

PreloadView.prototype = {
  _highlightHash: function() {
    var highlighted = document.getElementsByClassName('highlight');
    for (var i = 0; i < highlighted.length; i++) {
      highlighted[i].classList.remove('highlight');
    }

    var el = $(location.hash);
    if (el) {
      el.classList.add('highlight');
    }
  },

  _removeHash: function() {
    history.replaceState(
        {}, document.title, window.location.pathname + window.location.search);
  },

  _checkboxChangedHandler: function(ev) {
    $('#submit')
        .disabled =
        !($('#checkbox-owner').checked && $('#checkbox-subdomains').checked);
  },

  _clearTheme: function(theme) {
    document.body.classList.remove('theme-green', 'theme-yellow', 'theme-red');
  },

  setTheme: function(theme) {
    this._clearTheme();
    document.body.classList.add(theme)
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
    this._clearTheme();
    this.hideWaiting();
  },

  clickCheck() {
    $('#check').click();
  },

  setDomain(domain) {
    $('#domain').value = domain;
  },

  currentDomain: function() {
    var domain = $('#domain').value;
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

  _clearIssues: function() {
    $('#errors').textContent = '';
    $('#warnings').textContent = '';
  },

  showSubmission: function(domain) {
    $('#checkbox-owner').checked = false;
    $('#checkbox-subdomains').checked = false;
    $('#submit').disabled = true;

    var domainTexts = document.getElementsByClassName('domain-text');
    for (var i = 0; i < domainTexts.length; i++) {
      domainTexts[i].textContent = domain;
    }

    document.getElementById('oops-mailto').href =
        'mailto:hstspreload@chromium.org?subject=Domain%20with%20possible%20accidental%20preload:%20' +
        domain;
    document.getElementById('submit').value =
        'Submit ' + domain + ' to the HSTS preload list'

    $('#submit-form').classList.remove('hidden');
  },

  _hideSubmission: function() {
    $('#submit-success').hide();
    $('#submit-failure').hide();
    $('#ssl-labs-link').href = 'https://www.ssllabs.com/ssltest/analyze.html';
    $('#submit-form').hide();
  },

  clearSummary: function() { this.setSummary(''); },

  setSummary: function(summaryMessage) {
    $('#summary').textContent = summaryMessage;
  },

  clearStatus: function() { this.setStatus(''); },

  setStatus: function(statusMessage) {
    $('#status').textContent = statusMessage;
  },

  _createIssueElement: function(issue, type, typeLabel) {
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
  },

  showIssues: function(issues) {
    this._clearIssues();
    for (var e of issues.errors) {
      $('#errors').appendChild(this._createIssueElement(e, 'error', 'Error'));
    }
    for (var w of issues.warnings) {
      $('#warnings')
          .appendChild(this._createIssueElement(w, 'warning', 'Warning'));
    }
  }

};
