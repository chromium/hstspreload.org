'use strict';

var PreloadForm = function() {
  this._urlParam = new URLParam();
  this._view = new PreloadView(this.submitDomain.bind(this), this._urlParam);
  this._hstsPreload = new HSTSPreload();

  var domainParam = this._urlParam.get();
  if (domainParam) {
    var domain = extractDomain(domainParam);

    if (domain != domainParam) {
      this._view.setDomain(domain);
      this._view.clickCheck();
    } else {
      console.log('From URL parameter:', domain);
      $('#domain').value = domain;
      this.checkDomain(domain);
    }
  }
};

PreloadForm.prototype = {
  checkDomain: function(domain) {
    this._view.clearDomainSpecificElements();
    this._view.showWaiting(domain);

    if (!domain) {
      console.log('checkDomain called with empty domain');
      $('#result-waiting').classList.add('hidden');
      return;
    }

    this._currentResultsDomain = '';

    Promise
        .all([
          this._hstsPreload.status(domain),
          this._hstsPreload.preloadable(domain)
        ])
        .then(
            function(values) {
              this.handleResults(domain, values[0], values[1]);
            }.bind(this),
            function() {
              // TODO: handle failure better.
              $('#result').classList.remove('hidden');
              $('#result-waiting').classList.add('hidden');
            });
  },

  submitDomain: function() {
    var domain = this.domainToSubmit;

    this._hstsPreload.submit(domain).then(function(issues) {
      console.log('submit:', issues);
      if (issues.errors.length == 0) {
        $('#submit-success').show();
        $('#ssl-labs-link')
            .href = 'https://www.ssllabs.com/ssltest/analyze.html?d=' + domain;
      } else {
        this._view.setTheme('theme-red');
        $('#submit-failure').show();
        console.log(issues);
      }
    }.bind(this));
  },

  showResults: function(domain, issues, status) {
    var worstIssues;
    if (issues.errors.length == 0) {
      if (issues.warnings.length == 0) {
        worstIssues = 'none';
      } else {
        worstIssues = 'warnings';
      }
    } else {
      worstIssues = 'errors';
    }

    this._view.clearStatus();
    this._view.clearSummary();

    var generalElibigility = function() {
      switch (worstIssues) {
        case 'none':
          this._view.setSummary(
              'Eligibility: ' + domain +
              ' is eligible for the HSTS preload list.');
          this._view.setTheme('theme-green');
          break;
        case 'warnings':
          this._view.setSummary(
              'Eligibility: ' + domain +
              ' is eligible for preloading, although we recommend fixing the following warnings:');
          this._view.setTheme('theme-yellow');
          break;
        case 'errors':
          this._view.setSummary(
              'Eligibility: In order for ' + domain +
              ' to be elegible for preloading, the errors below must be resolved:');
          this._view.setTheme('theme-red');
          break;
      }

      this._view.showIssues(issues);
    }.bind(this);

    var pending = function() {
      switch (worstIssues) {
        case 'none':
          this._view.setStatus(
              'Status: ' + domain +
              ' is pending submission to the preload list.');
          this._view.setTheme('theme-green');
          break;
        case 'warnings':
          this._view.setStatus(
              'Status: ' + domain +
              ' is pending submission to the preload list.');
          this._view.setSummary(
              'However, it still has the following issues, which we recommend fixing:');
          this._view.setTheme('theme-yellow');
          this._view.showIssues(issues);
          break;
        case 'errors':
          this._view.setStatus(
              'Status: ' + domain +
              ' was recently submitted to the preload list.');
          this._view.setSummary(
              'However, ' + domain +
              ' has changed its behaviour since it was submitted, and will not be added to the official preload list unless the errors below are resolved:');
          this._view.setTheme('theme-red');
          this._view.showIssues(issues);
          break;
      }
    }.bind(this);

    switch (status.status) {
      case 'unknown':
        this._view.setStatus('Status: ' + domain + ' is not preloaded.');
        generalElibigility();
        break;

      case 'pending':
        pending();
        break;

      case 'preloaded':
        this._view.setStatus('Status: ' + domain + ' is currently preloaded.');
        this._view.setTheme('theme-green');
        break;

      case 'rejected':
        if (status.message) {
          this._view.setStatus(
              'Status: ' + domain +
              ' was previously rejected from the preload list for the following reason: ' +
              status.message);
        } else {
          this._view.setStatus(
              'Status: ' + domain +
              ' was previously rejected from the preload list.');
        }
        generalElibigility();
        break;

      case 'removed':
        this._view.setStatus(
            'Status: ' + domain +
            ' was previously on the preload list, but has been removed.');
        generalElibigility();
        break;

      default:
        this._view.setStatus('Cannot determine preload status.');
    }

    this._view.showResults()
  },

  handleResults: function(domain, status, issues) {
    console.log('handleResults:', status, issues);

    if (domain !== this._view.currentDomain()) {
      return;
    }
    if (this._currentResultsDomain === domain) {
      return;
    }
    this._currentResultsDomain = domain;

    if (domain !== this._view.currentDomain()) {
      console.log('Outdated result.');
      return;
    }

    this.showResults(domain, issues, status);
    this._view.hideWaiting();

    if (issues.errors.length === 0 &&
        ['unknown', 'rejected', 'removed'].indexOf(status.status) != -1) {
      this.domainToSubmit = domain;
      this._view.showSubmission(domain);
    }
  }
};

window.addEventListener('load', function() { new PreloadForm(); });
