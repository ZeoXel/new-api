'use client';

function _typeof(o) { "@babel/helpers - typeof"; return _typeof = "function" == typeof Symbol && "symbol" == typeof Symbol.iterator ? function (o) { return typeof o; } : function (o) { return o && "function" == typeof Symbol && o.constructor === Symbol && o !== Symbol.prototype ? "symbol" : typeof o; }, _typeof(o); }
var _excluded = ["size", "style"];
function ownKeys(e, r) { var t = Object.keys(e); if (Object.getOwnPropertySymbols) { var o = Object.getOwnPropertySymbols(e); r && (o = o.filter(function (r) { return Object.getOwnPropertyDescriptor(e, r).enumerable; })), t.push.apply(t, o); } return t; }
function _objectSpread(e) { for (var r = 1; r < arguments.length; r++) { var t = null != arguments[r] ? arguments[r] : {}; r % 2 ? ownKeys(Object(t), !0).forEach(function (r) { _defineProperty(e, r, t[r]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(e, Object.getOwnPropertyDescriptors(t)) : ownKeys(Object(t)).forEach(function (r) { Object.defineProperty(e, r, Object.getOwnPropertyDescriptor(t, r)); }); } return e; }
function _defineProperty(obj, key, value) { key = _toPropertyKey(key); if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _toPropertyKey(t) { var i = _toPrimitive(t, "string"); return "symbol" == _typeof(i) ? i : String(i); }
function _toPrimitive(t, r) { if ("object" != _typeof(t) || !t) return t; var e = t[Symbol.toPrimitive]; if (void 0 !== e) { var i = e.call(t, r || "default"); if ("object" != _typeof(i)) return i; throw new TypeError("@@toPrimitive must return a primitive value."); } return ("string" === r ? String : Number)(t); }
function _objectWithoutProperties(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }
function _objectWithoutPropertiesLoose(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
import { memo } from 'react';
import { TITLE } from "../style";
import { jsx as _jsx } from "react/jsx-runtime";
import { jsxs as _jsxs } from "react/jsx-runtime";
var Icon = /*#__PURE__*/memo(function (_ref) {
  var _ref$size = _ref.size,
    size = _ref$size === void 0 ? '1em' : _ref$size,
    style = _ref.style,
    rest = _objectWithoutProperties(_ref, _excluded);
  return /*#__PURE__*/_jsxs("svg", _objectSpread(_objectSpread({
    fill: "currentColor",
    fillRule: "evenodd",
    height: size,
    style: _objectSpread({
      flex: 'none',
      lineHeight: 1
    }, style),
    viewBox: "0 0 125 24",
    xmlns: "http://www.w3.org/2000/svg"
  }, rest), {}, {
    children: [/*#__PURE__*/_jsx("title", {
      children: TITLE
    }), /*#__PURE__*/_jsx("path", {
      d: "M2 12C2 5.8 5.94 2.329 11.699 2.329h6.914v3.693H11.92c-3.526 0-5.923 2.012-5.923 5.978 0 3.966 2.397 5.978 5.923 5.978h6.694v3.693H11.7C5.94 21.67 2 18.145 2 12zm20.194 3.5V2.331h3.857v12.37c0 2.396 1.24 3.25 3.278 3.25h2.313c2.012 0 3.278-.854 3.278-3.25V2.33h3.83v13.197c0 4.16-2.754 6.143-6.475 6.143h-3.581c-3.746 0-6.502-1.984-6.502-6.17h.002zM42.909 2.331h11.433c3.913 0 5.87 2.094 5.87 5.455 0 2.149-1.02 3.885-2.646 4.353 1.681.192 2.48 1.405 2.48 2.865v6.667h-3.884v-5.758c0-1.02-.304-1.736-1.68-1.736h-7.688v7.494H42.91V2.33zM53.82 10.57c1.764 0 2.48-.937 2.48-2.314 0-1.488-.715-2.285-2.533-2.285h-6.97v4.601h7.026l-.003-.002zm9.834 7.436h10.663c1.296 0 2.121-.716 2.121-2.012 0-1.35-.853-1.873-2.204-1.985l-5.374-.412c-3.388-.248-5.702-1.957-5.702-5.594 0-3.609 2.561-5.674 5.923-5.674h10.551v3.665H69.301c-1.488 0-2.26.716-2.26 1.984 0 1.321.827 1.873 2.288 2.012l5.454.385c3.333.248 5.538 2.012 5.538 5.565 0 3.417-2.341 5.73-5.73 5.73H63.65v-3.664h.003zm18.623-6.034c0-5.87 4.242-9.972 9.918-9.972h.056c5.675 0 9.947 4.105 9.947 9.972 0 5.895-4.27 10.028-9.947 10.028h-.056c-5.674 0-9.918-4.133-9.918-10.028zm9.946 6.282c3.498 0 6.062-2.48 6.062-6.254 0-3.746-2.561-6.254-6.062-6.254-3.472 0-6.034 2.508-6.034 6.254 0 3.774 2.562 6.254 6.034 6.254zm13.03-15.923h11.432c3.913 0 5.87 2.094 5.87 5.455 0 2.149-1.02 3.885-2.645 4.353 1.68.192 2.48 1.405 2.48 2.865v6.667h-3.885v-5.758c0-1.02-.304-1.736-1.681-1.736h-7.686v7.494h-3.885V2.33zm10.911 8.239c1.764 0 2.48-.937 2.48-2.314 0-1.488-.716-2.285-2.534-2.285h-6.97v4.601h7.026l-.002-.002z"
    })]
  }));
});
export default Icon;