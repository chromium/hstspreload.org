'use strict';

var $ = document.querySelector.bind(document);
Element.prototype.createChild = function(tagName, className) {
  var el = document.createElement(tagName);
  if (className) {
    el.classList.add(className);
  }
  this.appendChild(el);
  return el;
};

Element.prototype.show = function() {
  this.classList.remove('hidden');
};
Element.prototype.hide = function() {
  this.classList.add('hidden');
};

var URLParam = function() {};

URLParam.prototype = {
  get: function() {
    var match = window.location.search.match(/^\?domain=([^?&]+)$/);
    return match ? decodeURIComponent(match[1]) : null;
  }
};

// Strips away everything up to the first `://`.
// Returns the input if `://` doesn't appear.
var extractDomain = function(url) {
  if (url.indexOf('://') === -1) {
    url = 'https://' + url;
  }
  var a = document.createElement('a');
  a.href = url;
  return a.hostname;
};

var HSTSPreload = function() {};

HSTSPreload.prototype = {
  callAPI: function(method, endpoint, domain) {
    var path = '/api/v2/' + endpoint + '?domain=' + encodeURIComponent(domain);
    console.log('XHR:', path);
    // TODO: look at response codes.
    return new Promise(function(resolve, reject) {
      var req = new XMLHttpRequest();

      var onload = function(ev) { resolve(JSON.parse(req.response)); };

      req.addEventListener('load', onload);
      req.addEventListener(
          'error', (function(err) { reject(err); }).bind(this));
      req.open(method, path);
      req.send();
    });
  },

  status: function(domain) { return this.callAPI('GET', 'status', domain); },

  preloadable: function(domain) {
    return this.callAPI('GET', 'preloadable', domain);
  },

  submit: function(domain) { return this.callAPI('POST', 'submit', domain); }
};
