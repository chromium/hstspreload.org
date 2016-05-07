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

var URLParam = function() {};

URLParam.prototype = {
  get: function() {
    var match = window.location.search.match(/^\?domain=([^?&]+)$/);
    return match ? decodeURIComponent(match[1]) : null;
  }
};

var HSTSPreload = function() {}

                  HSTSPreload.prototype = {
  callAPI: function(endpoint, domain) {
    var path = '/' + endpoint + '?domain=' + encodeURIComponent(domain);
    console.log('XHR:', path);
    // TODO: look at response codes.
    return new Promise(function(resolve, reject) {
      var req = new XMLHttpRequest();

      var onload = function(ev) { resolve(JSON.parse(req.response)); };

      req.addEventListener('load', onload);
      req.addEventListener(
          'error', (function(err) { reject(err); }).bind(this));
      req.open('GET', path);
      req.send();
    });
  },

  status: function(domain) { return this.callAPI('status', domain); },

  preloadable: function(domain) { return this.callAPI('preloadable', domain); },

  submit: function(domain) { return this.callAPI('submit', domain); }
};
