(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[460],{4184:function(e,t){var r;!function(){"use strict";var n={}.hasOwnProperty;function o(){for(var e=[],t=0;t<arguments.length;t++){var r=arguments[t];if(r){var a=typeof r;if("string"===a||"number"===a)e.push(r);else if(Array.isArray(r)){if(r.length){var u=o.apply(null,r);u&&e.push(u)}}else if("object"===a)if(r.toString===Object.prototype.toString)for(var l in r)n.call(r,l)&&r[l]&&e.push(l);else e.push(r.toString())}}return e.join(" ")}e.exports?(o.default=o,e.exports=o):void 0===(r=function(){return o}.apply(t,[]))||(e.exports=r)}()},1551:function(e,t,r){"use strict";function n(e,t){(null==t||t>e.length)&&(t=e.length);for(var r=0,n=new Array(t);r<t;r++)n[r]=e[r];return n}function o(e,t){return function(e){if(Array.isArray(e))return e}(e)||function(e,t){var r=null==e?null:"undefined"!==typeof Symbol&&e[Symbol.iterator]||e["@@iterator"];if(null!=r){var n,o,a=[],u=!0,l=!1;try{for(r=r.call(e);!(u=(n=r.next()).done)&&(a.push(n.value),!t||a.length!==t);u=!0);}catch(i){l=!0,o=i}finally{try{u||null==r.return||r.return()}finally{if(l)throw o}}return a}}(e,t)||function(e,t){if(!e)return;if("string"===typeof e)return n(e,t);var r=Object.prototype.toString.call(e).slice(8,-1);"Object"===r&&e.constructor&&(r=e.constructor.name);if("Map"===r||"Set"===r)return Array.from(r);if("Arguments"===r||/^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(r))return n(e,t)}(e,t)||function(){throw new TypeError("Invalid attempt to destructure non-iterable instance.\\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method.")}()}Object.defineProperty(t,"__esModule",{value:!0}),t.default=void 0;var a,u=(a=r(7294))&&a.__esModule?a:{default:a},l=r(1003),i=r(880),f=r(9246);var c={};function s(e,t,r,n){if(e&&l.isLocalURL(t)){e.prefetch(t,r,n).catch((function(e){0}));var o=n&&"undefined"!==typeof n.locale?n.locale:e&&e.locale;c[t+"%"+r+(o?"%"+o:"")]=!0}}var p=function(e){var t,r=!1!==e.prefetch,n=i.useRouter(),a=u.default.useMemo((function(){var t=o(l.resolveHref(n,e.href,!0),2),r=t[0],a=t[1];return{href:r,as:e.as?l.resolveHref(n,e.as):a||r}}),[n,e.href,e.as]),p=a.href,d=a.as,v=u.default.useRef(p),y=u.default.useRef(d),h=e.children,b=e.replace,g=e.shallow,m=e.scroll,w=e.locale;"string"===typeof h&&(h=u.default.createElement("a",null,h));var j=(t=u.default.Children.only(h))&&"object"===typeof t&&t.ref,E=o(f.useIntersection({rootMargin:"200px"}),3),M=E[0],k=E[1],L=E[2],A=u.default.useCallback((function(e){y.current===d&&v.current===p||(L(),y.current=d,v.current=p),M(e),j&&("function"===typeof j?j(e):"object"===typeof j&&(j.current=e))}),[d,j,p,L,M]);u.default.useEffect((function(){var e=k&&r&&l.isLocalURL(p),t="undefined"!==typeof w?w:n&&n.locale,o=c[p+"%"+d+(t?"%"+t:"")];e&&!o&&s(n,p,d,{locale:t})}),[d,p,k,w,r,n]);var C={ref:A,onClick:function(e){t.props&&"function"===typeof t.props.onClick&&t.props.onClick(e),e.defaultPrevented||function(e,t,r,n,o,a,u,i){("A"!==e.currentTarget.nodeName.toUpperCase()||!function(e){var t=e.currentTarget.target;return t&&"_self"!==t||e.metaKey||e.ctrlKey||e.shiftKey||e.altKey||e.nativeEvent&&2===e.nativeEvent.which}(e)&&l.isLocalURL(r))&&(e.preventDefault(),t[o?"replace":"push"](r,n,{shallow:a,locale:i,scroll:u}))}(e,n,p,d,b,g,m,w)},onMouseEnter:function(e){t.props&&"function"===typeof t.props.onMouseEnter&&t.props.onMouseEnter(e),l.isLocalURL(p)&&s(n,p,d,{priority:!0})}};if(e.passHref||"a"===t.type&&!("href"in t.props)){var S="undefined"!==typeof w?w:n&&n.locale,I=n&&n.isLocaleDomain&&l.getDomainLocale(d,S,n&&n.locales,n&&n.domainLocales);C.href=I||l.addBasePath(l.addLocale(d,S,n&&n.defaultLocale))}return u.default.cloneElement(t,C)};t.default=p,("function"===typeof t.default||"object"===typeof t.default&&null!==t.default)&&(Object.assign(t.default,t),e.exports=t.default)},9246:function(e,t,r){"use strict";function n(e,t){(null==t||t>e.length)&&(t=e.length);for(var r=0,n=new Array(t);r<t;r++)n[r]=e[r];return n}function o(e,t){return function(e){if(Array.isArray(e))return e}(e)||function(e,t){var r=null==e?null:"undefined"!==typeof Symbol&&e[Symbol.iterator]||e["@@iterator"];if(null!=r){var n,o,a=[],u=!0,l=!1;try{for(r=r.call(e);!(u=(n=r.next()).done)&&(a.push(n.value),!t||a.length!==t);u=!0);}catch(i){l=!0,o=i}finally{try{u||null==r.return||r.return()}finally{if(l)throw o}}return a}}(e,t)||function(e,t){if(!e)return;if("string"===typeof e)return n(e,t);var r=Object.prototype.toString.call(e).slice(8,-1);"Object"===r&&e.constructor&&(r=e.constructor.name);if("Map"===r||"Set"===r)return Array.from(r);if("Arguments"===r||/^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(r))return n(e,t)}(e,t)||function(){throw new TypeError("Invalid attempt to destructure non-iterable instance.\\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method.")}()}Object.defineProperty(t,"__esModule",{value:!0}),t.useIntersection=function(e){var t=e.rootRef,r=e.rootMargin,n=e.disabled||!l,c=a.useRef(),s=o(a.useState(!1),2),p=s[0],d=s[1],v=o(a.useState(t?t.current:null),2),y=v[0],h=v[1],b=a.useCallback((function(e){c.current&&(c.current(),c.current=void 0),n||p||e&&e.tagName&&(c.current=function(e,t,r){var n=function(e){var t,r={root:e.root||null,margin:e.rootMargin||""},n=f.find((function(e){return e.root===r.root&&e.margin===r.margin}));n?t=i.get(n):(t=i.get(r),f.push(r));if(t)return t;var o=new Map,a=new IntersectionObserver((function(e){e.forEach((function(e){var t=o.get(e.target),r=e.isIntersecting||e.intersectionRatio>0;t&&r&&t(r)}))}),e);return i.set(r,t={id:r,observer:a,elements:o}),t}(r),o=n.id,a=n.observer,u=n.elements;return u.set(e,t),a.observe(e),function(){if(u.delete(e),a.unobserve(e),0===u.size){a.disconnect(),i.delete(o);var t=f.findIndex((function(e){return e.root===o.root&&e.margin===o.margin}));t>-1&&f.splice(t,1)}}}(e,(function(e){return e&&d(e)}),{root:y,rootMargin:r}))}),[n,y,r,p]),g=a.useCallback((function(){d(!1)}),[]);return a.useEffect((function(){if(!l&&!p){var e=u.requestIdleCallback((function(){return d(!0)}));return function(){return u.cancelIdleCallback(e)}}}),[p]),a.useEffect((function(){t&&h(t.current)}),[t]),[b,p,g]};var a=r(7294),u=r(4686),l="undefined"!==typeof IntersectionObserver;var i=new Map,f=[];("function"===typeof t.default||"object"===typeof t.default&&null!==t.default)&&(Object.assign(t.default,t),e.exports=t.default)},1664:function(e,t,r){e.exports=r(1551)},5506:function(e,t,r){"use strict";var n=r(7294);const o=n.forwardRef((function(e,t){return n.createElement("svg",Object.assign({xmlns:"http://www.w3.org/2000/svg",fill:"none",viewBox:"0 0 24 24",strokeWidth:2,stroke:"currentColor","aria-hidden":"true",ref:t},e),n.createElement("path",{strokeLinecap:"round",strokeLinejoin:"round",d:"M6 18L18 6M6 6l12 12"}))}));t.Z=o}}]);