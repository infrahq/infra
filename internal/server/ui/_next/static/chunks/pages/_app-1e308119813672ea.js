(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[888],{7484:function(e){e.exports=function(){"use strict";var e=1e3,t=6e4,n=36e5,r="millisecond",i="second",u="minute",o="hour",a="day",s="week",c="month",f="quarter",l="year",d="date",h="Invalid Date",v=/^(\d{4})[-/]?(\d{1,2})?[-/]?(\d{0,2})[Tt\s]*(\d{1,2})?:?(\d{1,2})?:?(\d{1,2})?[.:]?(\d+)?$/,p=/\[([^\]]+)]|Y{1,4}|M{1,4}|D{1,2}|d{1,4}|H{1,2}|h{1,2}|a|A|m{1,2}|s{1,2}|Z{1,2}|SSS/g,y={name:"en",weekdays:"Sunday_Monday_Tuesday_Wednesday_Thursday_Friday_Saturday".split("_"),months:"January_February_March_April_May_June_July_August_September_October_November_December".split("_")},g=function(e,t,n){var r=String(e);return!r||r.length>=t?e:""+Array(t+1-r.length).join(n)+e},b={s:g,z:function(e){var t=-e.utcOffset(),n=Math.abs(t),r=Math.floor(n/60),i=n%60;return(t<=0?"+":"-")+g(r,2,"0")+":"+g(i,2,"0")},m:function e(t,n){if(t.date()<n.date())return-e(n,t);var r=12*(n.year()-t.year())+(n.month()-t.month()),i=t.clone().add(r,c),u=n-i<0,o=t.clone().add(r+(u?-1:1),c);return+(-(r+(n-i)/(u?i-o:o-i))||0)},a:function(e){return e<0?Math.ceil(e)||0:Math.floor(e)},p:function(e){return{M:c,y:l,w:s,d:a,D:d,h:o,m:u,s:i,ms:r,Q:f}[e]||String(e||"").toLowerCase().replace(/s$/,"")},u:function(e){return void 0===e}},m="en",w={};w[m]=y;var O=function(e){return e instanceof M},$=function e(t,n,r){var i;if(!t)return m;if("string"==typeof t){var u=t.toLowerCase();w[u]&&(i=u),n&&(w[u]=n,i=u);var o=t.split("-");if(!i&&o.length>1)return e(o[0])}else{var a=t.name;w[a]=t,i=a}return!r&&i&&(m=i),i||!r&&m},_=function(e,t){if(O(e))return e.clone();var n="object"==typeof t?t:{};return n.date=e,n.args=arguments,new M(n)},S=b;S.l=$,S.i=O,S.w=function(e,t){return _(e,{locale:t.$L,utc:t.$u,x:t.$x,$offset:t.$offset})};var M=function(){function y(e){this.$L=$(e.locale,null,!0),this.parse(e)}var g=y.prototype;return g.parse=function(e){this.$d=function(e){var t=e.date,n=e.utc;if(null===t)return new Date(NaN);if(S.u(t))return new Date;if(t instanceof Date)return new Date(t);if("string"==typeof t&&!/Z$/i.test(t)){var r=t.match(v);if(r){var i=r[2]-1||0,u=(r[7]||"0").substring(0,3);return n?new Date(Date.UTC(r[1],i,r[3]||1,r[4]||0,r[5]||0,r[6]||0,u)):new Date(r[1],i,r[3]||1,r[4]||0,r[5]||0,r[6]||0,u)}}return new Date(t)}(e),this.$x=e.x||{},this.init()},g.init=function(){var e=this.$d;this.$y=e.getFullYear(),this.$M=e.getMonth(),this.$D=e.getDate(),this.$W=e.getDay(),this.$H=e.getHours(),this.$m=e.getMinutes(),this.$s=e.getSeconds(),this.$ms=e.getMilliseconds()},g.$utils=function(){return S},g.isValid=function(){return!(this.$d.toString()===h)},g.isSame=function(e,t){var n=_(e);return this.startOf(t)<=n&&n<=this.endOf(t)},g.isAfter=function(e,t){return _(e)<this.startOf(t)},g.isBefore=function(e,t){return this.endOf(t)<_(e)},g.$g=function(e,t,n){return S.u(e)?this[t]:this.set(n,e)},g.unix=function(){return Math.floor(this.valueOf()/1e3)},g.valueOf=function(){return this.$d.getTime()},g.startOf=function(e,t){var n=this,r=!!S.u(t)||t,f=S.p(e),h=function(e,t){var i=S.w(n.$u?Date.UTC(n.$y,t,e):new Date(n.$y,t,e),n);return r?i:i.endOf(a)},v=function(e,t){return S.w(n.toDate()[e].apply(n.toDate("s"),(r?[0,0,0,0]:[23,59,59,999]).slice(t)),n)},p=this.$W,y=this.$M,g=this.$D,b="set"+(this.$u?"UTC":"");switch(f){case l:return r?h(1,0):h(31,11);case c:return r?h(1,y):h(0,y+1);case s:var m=this.$locale().weekStart||0,w=(p<m?p+7:p)-m;return h(r?g-w:g+(6-w),y);case a:case d:return v(b+"Hours",0);case o:return v(b+"Minutes",1);case u:return v(b+"Seconds",2);case i:return v(b+"Milliseconds",3);default:return this.clone()}},g.endOf=function(e){return this.startOf(e,!1)},g.$set=function(e,t){var n,s=S.p(e),f="set"+(this.$u?"UTC":""),h=(n={},n[a]=f+"Date",n[d]=f+"Date",n[c]=f+"Month",n[l]=f+"FullYear",n[o]=f+"Hours",n[u]=f+"Minutes",n[i]=f+"Seconds",n[r]=f+"Milliseconds",n)[s],v=s===a?this.$D+(t-this.$W):t;if(s===c||s===l){var p=this.clone().set(d,1);p.$d[h](v),p.init(),this.$d=p.set(d,Math.min(this.$D,p.daysInMonth())).$d}else h&&this.$d[h](v);return this.init(),this},g.set=function(e,t){return this.clone().$set(e,t)},g.get=function(e){return this[S.p(e)]()},g.add=function(r,f){var d,h=this;r=Number(r);var v=S.p(f),p=function(e){var t=_(h);return S.w(t.date(t.date()+Math.round(e*r)),h)};if(v===c)return this.set(c,this.$M+r);if(v===l)return this.set(l,this.$y+r);if(v===a)return p(1);if(v===s)return p(7);var y=(d={},d[u]=t,d[o]=n,d[i]=e,d)[v]||1,g=this.$d.getTime()+r*y;return S.w(g,this)},g.subtract=function(e,t){return this.add(-1*e,t)},g.format=function(e){var t=this,n=this.$locale();if(!this.isValid())return n.invalidDate||h;var r=e||"YYYY-MM-DDTHH:mm:ssZ",i=S.z(this),u=this.$H,o=this.$m,a=this.$M,s=n.weekdays,c=n.months,f=function(e,n,i,u){return e&&(e[n]||e(t,r))||i[n].slice(0,u)},l=function(e){return S.s(u%12||12,e,"0")},d=n.meridiem||function(e,t,n){var r=e<12?"AM":"PM";return n?r.toLowerCase():r},v={YY:String(this.$y).slice(-2),YYYY:this.$y,M:a+1,MM:S.s(a+1,2,"0"),MMM:f(n.monthsShort,a,c,3),MMMM:f(c,a),D:this.$D,DD:S.s(this.$D,2,"0"),d:String(this.$W),dd:f(n.weekdaysMin,this.$W,s,2),ddd:f(n.weekdaysShort,this.$W,s,3),dddd:s[this.$W],H:String(u),HH:S.s(u,2,"0"),h:l(1),hh:l(2),a:d(u,o,!0),A:d(u,o,!1),m:String(o),mm:S.s(o,2,"0"),s:String(this.$s),ss:S.s(this.$s,2,"0"),SSS:S.s(this.$ms,3,"0"),Z:i};return r.replace(p,(function(e,t){return t||v[e]||i.replace(":","")}))},g.utcOffset=function(){return 15*-Math.round(this.$d.getTimezoneOffset()/15)},g.diff=function(r,d,h){var v,p=S.p(d),y=_(r),g=(y.utcOffset()-this.utcOffset())*t,b=this-y,m=S.m(this,y);return m=(v={},v[l]=m/12,v[c]=m,v[f]=m/3,v[s]=(b-g)/6048e5,v[a]=(b-g)/864e5,v[o]=b/n,v[u]=b/t,v[i]=b/e,v)[p]||b,h?m:S.a(m)},g.daysInMonth=function(){return this.endOf(c).$D},g.$locale=function(){return w[this.$L]},g.locale=function(e,t){if(!e)return this.$L;var n=this.clone(),r=$(e,t,!0);return r&&(n.$L=r),n},g.clone=function(){return S.w(this.$d,this)},g.toDate=function(){return new Date(this.valueOf())},g.toJSON=function(){return this.isValid()?this.toISOString():null},g.toISOString=function(){return this.$d.toISOString()},g.toString=function(){return this.$d.toUTCString()},y}(),D=M.prototype;return _.prototype=D,[["$ms",r],["$s",i],["$m",u],["$H",o],["$W",a],["$M",c],["$y",l],["$D",d]].forEach((function(e){D[e[1]]=function(t){return this.$g(t,e[0],e[1])}})),_.extend=function(e,t){return e.$i||(e(t,M,_),e.$i=!0),_},_.locale=$,_.isDayjs=O,_.unix=function(e){return _(1e3*e)},_.en=w[m],_.Ls=w,_.p={},_}()},4110:function(e){e.exports=function(){"use strict";return function(e,t,n){e=e||{};var r=t.prototype,i={future:"in %s",past:"%s ago",s:"a few seconds",m:"a minute",mm:"%d minutes",h:"an hour",hh:"%d hours",d:"a day",dd:"%d days",M:"a month",MM:"%d months",y:"a year",yy:"%d years"};function u(e,t,n,i){return r.fromToBase(e,t,n,i)}n.en.relativeTime=i,r.fromToBase=function(t,r,u,o,a){for(var s,c,f,l=u.$locale().relativeTime||i,d=e.thresholds||[{l:"s",r:44,d:"second"},{l:"m",r:89},{l:"mm",r:44,d:"minute"},{l:"h",r:89},{l:"hh",r:21,d:"hour"},{l:"d",r:35},{l:"dd",r:25,d:"day"},{l:"M",r:45},{l:"MM",r:10,d:"month"},{l:"y",r:17},{l:"yy",d:"year"}],h=d.length,v=0;v<h;v+=1){var p=d[v];p.d&&(s=o?n(t).diff(u,p.d,!0):u.diff(t,p.d,!0));var y=(e.rounding||Math.round)(Math.abs(s));if(f=s>0,y<=p.r||!p.r){y<=1&&v>0&&(p=d[v-1]);var g=l[p.l];a&&(y=a(""+y)),c="string"==typeof g?g.replace("%d",y):g(y,r,p.l,f);break}}if(r)return c;var b=f?l.future:l.past;return"function"==typeof b?b(c):b.replace("%s",c)},r.to=function(e,t){return u(e,t,this,!0)},r.from=function(e,t){return u(e,t,this)};var o=function(e){return e.$u?n.utc():n()};r.toNow=function(e){return this.to(o(this),e)},r.fromNow=function(e){return this.from(o(this),e)}}}()},660:function(e){e.exports=function(){"use strict";return function(e,t,n){n.updateLocale=function(e,t){var r=n.Ls[e];if(r)return(t?Object.keys(t):[]).forEach((function(e){r[e]=t[e]})),r}}}()},1780:function(e,t,n){(window.__NEXT_P=window.__NEXT_P||[]).push(["/_app",function(){return n(5635)}])},7645:function(e,t,n){"use strict";function r(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function i(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{},i=Object.keys(n);"function"===typeof Object.getOwnPropertySymbols&&(i=i.concat(Object.getOwnPropertySymbols(n).filter((function(e){return Object.getOwnPropertyDescriptor(n,e).enumerable})))),i.forEach((function(t){r(e,t,n[t])}))}return e}t.default=function(e,t){var n=u.default,r={loading:function(e){e.error,e.isLoading;return e.pastDelay,null}};o=e,s=Promise,(null!=s&&"undefined"!==typeof Symbol&&s[Symbol.hasInstance]?s[Symbol.hasInstance](o):o instanceof s)?r.loader=function(){return e}:"function"===typeof e?r.loader=e:"object"===typeof e&&(r=i({},r,e));var o,s;var c=r=i({},r,t);if(c.suspense)throw new Error("Invalid suspense option usage in next/dynamic. Read more: https://nextjs.org/docs/messages/invalid-dynamic-suspense");if(c.suspense)return n(c);r.loadableGenerated&&delete(r=i({},r,r.loadableGenerated)).loadableGenerated;if("boolean"===typeof r.ssr){if(!r.ssr)return delete r.ssr,a(n,r);delete r.ssr}return n(r)};o(n(7294));var u=o(n(4588));function o(e){return e&&e.__esModule?e:{default:e}}function a(e,t){return delete t.webpack,delete t.modules,e(t)}},3644:function(e,t,n){"use strict";var r;Object.defineProperty(t,"__esModule",{value:!0}),t.LoadableContext=void 0;var i=((r=n(7294))&&r.__esModule?r:{default:r}).default.createContext(null);t.LoadableContext=i},4588:function(e,t,n){"use strict";function r(e,t){for(var n=0;n<t.length;n++){var r=t[n];r.enumerable=r.enumerable||!1,r.configurable=!0,"value"in r&&(r.writable=!0),Object.defineProperty(e,r.key,r)}}function i(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function u(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{},r=Object.keys(n);"function"===typeof Object.getOwnPropertySymbols&&(r=r.concat(Object.getOwnPropertySymbols(n).filter((function(e){return Object.getOwnPropertyDescriptor(n,e).enumerable})))),r.forEach((function(t){i(e,t,n[t])}))}return e}Object.defineProperty(t,"__esModule",{value:!0}),t.default=void 0;var o,a=(o=n(7294))&&o.__esModule?o:{default:o},s=n(2021),c=n(3644);var f=[],l=[],d=!1;function h(e){var t=e(),n={loading:!0,loaded:null,error:null};return n.promise=t.then((function(e){return n.loading=!1,n.loaded=e,e})).catch((function(e){throw n.loading=!1,n.error=e,e})),n}var v=function(){function e(t,n){!function(e,t){if(!(e instanceof t))throw new TypeError("Cannot call a class as a function")}(this,e),this._loadFn=t,this._opts=n,this._callbacks=new Set,this._delay=null,this._timeout=null,this.retry()}var t,n,i;return t=e,(n=[{key:"promise",value:function(){return this._res.promise}},{key:"retry",value:function(){var e=this;this._clearTimeouts(),this._res=this._loadFn(this._opts.loader),this._state={pastDelay:!1,timedOut:!1};var t=this._res,n=this._opts;if(t.loading){if("number"===typeof n.delay)if(0===n.delay)this._state.pastDelay=!0;else{var r=this;this._delay=setTimeout((function(){r._update({pastDelay:!0})}),n.delay)}if("number"===typeof n.timeout){var i=this;this._timeout=setTimeout((function(){i._update({timedOut:!0})}),n.timeout)}}this._res.promise.then((function(){e._update({}),e._clearTimeouts()})).catch((function(t){e._update({}),e._clearTimeouts()})),this._update({})}},{key:"_update",value:function(e){this._state=u({},this._state,{error:this._res.error,loaded:this._res.loaded,loading:this._res.loading},e),this._callbacks.forEach((function(e){return e()}))}},{key:"_clearTimeouts",value:function(){clearTimeout(this._delay),clearTimeout(this._timeout)}},{key:"getCurrentValue",value:function(){return this._state}},{key:"subscribe",value:function(e){var t=this;return this._callbacks.add(e),function(){t._callbacks.delete(e)}}}])&&r(t.prototype,n),i&&r(t,i),e}();function p(e){return function(e,t){var n=function(){if(!i){var t=new v(e,r);i={getCurrentValue:t.getCurrentValue.bind(t),subscribe:t.subscribe.bind(t),retry:t.retry.bind(t),promise:t.promise.bind(t)}}return i.promise()},r=Object.assign({loader:null,loading:null,delay:200,timeout:null,webpack:null,modules:null,suspense:!1},t);r.suspense&&(r.lazy=a.default.lazy(r.loader));var i=null;if(!d&&!r.suspense){var o=r.webpack?r.webpack():r.modules;o&&l.push((function(e){var t=!0,r=!1,i=void 0;try{for(var u,a=o[Symbol.iterator]();!(t=(u=a.next()).done);t=!0){var s=u.value;if(-1!==e.indexOf(s))return n()}}catch(c){r=!0,i=c}finally{try{t||null==a.return||a.return()}finally{if(r)throw i}}}))}var f=r.suspense?function(e,t){return a.default.createElement(r.lazy,u({},e,{ref:t}))}:function(e,t){n();var u=a.default.useContext(c.LoadableContext),o=s.useSubscription(i);return a.default.useImperativeHandle(t,(function(){return{retry:i.retry}}),[]),u&&Array.isArray(r.modules)&&r.modules.forEach((function(e){u(e)})),a.default.useMemo((function(){return o.loading||o.error?a.default.createElement(r.loading,{isLoading:o.loading,pastDelay:o.pastDelay,timedOut:o.timedOut,error:o.error,retry:i.retry}):o.loaded?a.default.createElement(function(e){return e&&e.__esModule?e.default:e}(o.loaded),e):null}),[e,o])};return f.preload=function(){return!r.suspense&&n()},f.displayName="LoadableComponent",a.default.forwardRef(f)}(h,e)}function y(e,t){for(var n=[];e.length;){var r=e.pop();n.push(r(t))}return Promise.all(n).then((function(){if(e.length)return y(e,t)}))}p.preloadAll=function(){return new Promise((function(e,t){y(f).then(e,t)}))},p.preloadReady=function(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:[];return new Promise((function(t){var n=function(){return d=!0,t()};y(l,e).then(n,n)}))},window.__NEXT_PRELOADREADY=p.preloadReady;var g=p;t.default=g},5635:function(e,t,n){"use strict";n.r(t),n.d(t,{default:function(){return x}});var r,i,u=n(4051),o=n.n(u),a=n(5893),s=n(5152),c=n(1163),f=n(9008),l=n(8100),d=function(){return d=Object.assign||function(e){for(var t,n=1,r=arguments.length;n<r;n++)for(var i in t=arguments[n])Object.prototype.hasOwnProperty.call(t,i)&&(e[i]=t[i]);return e},d.apply(this,arguments)},h=function(e){return"function"==typeof e[1]?[e[0],e[1],e[2]||{}]:[e[0],null,(null===e[1]?e[2]:e[1])||{}]},v=(r=l.ZP,i=function(e){return function(t,n,r){return r.revalidateOnFocus=!1,r.revalidateIfStale=!1,r.revalidateOnReconnect=!1,e(t,n,r)}},function(){for(var e=[],t=0;t<arguments.length;t++)e[t]=arguments[t];var n=h(e),u=n[0],o=n[1],a=n[2],s=(a.use||[]).concat(i);return r(u,o,d(d({},a),{use:s}))});function p(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}var y=n.g.fetch;function g(e){return function(t,n){return y(t,function(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{},r=Object.keys(n);"function"===typeof Object.getOwnPropertySymbols&&(r=r.concat(Object.getOwnPropertySymbols(n).filter((function(e){return Object.getOwnPropertyDescriptor(n,e).enumerable})))),r.forEach((function(t){p(e,t,n[t])}))}return e}({},n,t.startsWith("/")?{headers:{"Infra-Version":e}}:{}))}}n.g.fetch=g("0.12.0");var b=n(4110),m=n.n(b),w=n(660),O=n.n(w),$=n(7484),_=n.n($);_().extend(m()),_().extend(O()),_().updateLocale("en",{relativeTime:{future:"in %s",past:"%s",s:"just now",m:"a minute ago",mm:"%d minutes ago",h:"an hour ago",hh:"%d hours ago",d:"a day ago",dd:"%d days ago",M:"a month ago",MM:"%d months ago",y:"a year ago",yy:"%d years ago"}});n(6774);function S(e,t,n,r,i,u,o){try{var a=e[u](o),s=a.value}catch(c){return void n(c)}a.done?t(s):Promise.resolve(s).then(r,i)}function M(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function D(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{},r=Object.keys(n);"function"===typeof Object.getOwnPropertySymbols&&(r=r.concat(Object.getOwnPropertySymbols(n).filter((function(e){return Object.getOwnPropertyDescriptor(n,e).enumerable})))),r.forEach((function(t){M(e,t,n[t])}))}return e}var j=function(){var e,t=(e=o().mark((function e(t,n){var r,i;return o().wrap((function(e){for(;;)switch(e.prev=e.next){case 0:return e.next=2,fetch(t,D({},n,{headers:{"Infra-Version":"0.12.2"}}));case 2:return r=e.sent,e.next=5,r.json();case 5:if(i=e.sent,r.ok){e.next=8;break}throw i;case 8:return e.abrupt("return",i);case 9:case"end":return e.stop()}}),e)})),function(){var t=this,n=arguments;return new Promise((function(r,i){var u=e.apply(t,n);function o(e){S(u,r,i,o,a,"next",e)}function a(e){S(u,r,i,o,a,"throw",e)}o(void 0)}))});return function(e,n){return t.apply(this,arguments)}}();function k(e){var t=e.Component,n=e.pageProps,r=v("/v1/identities/self",j),i=r.data,u=r.error,o=v("/v1/signup",j),s=o.data,d=o.error,h=(0,c.useRouter)();if(!i&&!u||!s&&!d)return null;if((null===s||void 0===s?void 0:s.enabled)&&!h.asPath.startsWith("/signup"))return h.replace("/signup"),null;if(!(null===s||void 0===s?void 0:s.enabled)&&!i&&!h.asPath.startsWith("/login"))return h.replace("/login"),null;if((null===i||void 0===i?void 0:i.id)&&(h.asPath.startsWith("/login")||h.asPath.startsWith("/signup")))return h.replace("/"),null;var p=t.layout||function(e){return e};return(0,a.jsxs)(l.J$,{value:{fetcher:function(e,t){return j(e,t)},revalidateOnFocus:!1,revalidateOnReconnect:!1},children:[(0,a.jsxs)(f.default,{children:[(0,a.jsx)("link",{rel:"icon",type:"image/png",sizes:"32x32",href:"/favicon-32x32.png"}),(0,a.jsx)("link",{rel:"icon",type:"image/png",sizes:"16x16",href:"/favicon-16x16.png"}),(0,a.jsx)("title",{children:"Infra"})]}),p((0,a.jsx)(t,D({},n)))]})}var x=(0,s.default)((function(){return Promise.resolve(k)}),{ssr:!1})},6774:function(){},2021:function(e,t,n){(()=>{"use strict";var t={800:e=>{var t=Object.getOwnPropertySymbols,n=Object.prototype.hasOwnProperty,r=Object.prototype.propertyIsEnumerable;function i(e){if(null===e||void 0===e)throw new TypeError("Object.assign cannot be called with null or undefined");return Object(e)}e.exports=function(){try{if(!Object.assign)return!1;var e=new String("abc");if(e[5]="de","5"===Object.getOwnPropertyNames(e)[0])return!1;for(var t={},n=0;n<10;n++)t["_"+String.fromCharCode(n)]=n;var r=Object.getOwnPropertyNames(t).map((function(e){return t[e]}));if("0123456789"!==r.join(""))return!1;var i={};return"abcdefghijklmnopqrst".split("").forEach((function(e){i[e]=e})),"abcdefghijklmnopqrst"===Object.keys(Object.assign({},i)).join("")}catch(e){return!1}}()?Object.assign:function(e,u){for(var o,a,s=i(e),c=1;c<arguments.length;c++){for(var f in o=Object(arguments[c]))n.call(o,f)&&(s[f]=o[f]);if(t){a=t(o);for(var l=0;l<a.length;l++)r.call(o,a[l])&&(s[a[l]]=o[a[l]])}}return s}},569:(e,t,n)=>{0},403:(e,t,n)=>{var r=n(800),i=n(522);t.useSubscription=function(e){var t=e.getCurrentValue,n=e.subscribe,u=i.useState((function(){return{getCurrentValue:t,subscribe:n,value:t()}}));e=u[0];var o=u[1];return u=e.value,e.getCurrentValue===t&&e.subscribe===n||(u=t(),o({getCurrentValue:t,subscribe:n,value:u})),i.useDebugValue(u),i.useEffect((function(){function e(){if(!i){var e=t();o((function(i){return i.getCurrentValue!==t||i.subscribe!==n||i.value===e?i:r({},i,{value:e})}))}}var i=!1,u=n(e);return e(),function(){i=!0,u()}}),[t,n]),u}},138:(e,t,n)=>{e.exports=n(403)},522:e=>{e.exports=n(7294)}},r={};function i(e){var n=r[e];if(void 0!==n)return n.exports;var u=r[e]={exports:{}},o=!0;try{t[e](u,u.exports,i),o=!1}finally{o&&delete r[e]}return u.exports}i.ab="//";var u=i(138);e.exports=u})()},5152:function(e,t,n){e.exports=n(7645)},9008:function(e,t,n){e.exports=n(3121)},1163:function(e,t,n){e.exports=n(880)},8100:function(e,t,n){"use strict";n.d(t,{J$:function(){return B},ZP:function(){return X},kY:function(){return q}});var r=n(7294);function i(e,t,n,r){return new(n||(n=Promise))((function(i,u){function o(e){try{s(r.next(e))}catch(t){u(t)}}function a(e){try{s(r.throw(e))}catch(t){u(t)}}function s(e){var t;e.done?i(e.value):(t=e.value,t instanceof n?t:new n((function(e){e(t)}))).then(o,a)}s((r=r.apply(e,t||[])).next())}))}function u(e,t){var n,r,i,u,o={label:0,sent:function(){if(1&i[0])throw i[1];return i[1]},trys:[],ops:[]};return u={next:a(0),throw:a(1),return:a(2)},"function"===typeof Symbol&&(u[Symbol.iterator]=function(){return this}),u;function a(u){return function(a){return function(u){if(n)throw new TypeError("Generator is already executing.");for(;o;)try{if(n=1,r&&(i=2&u[0]?r.return:u[0]?r.throw||((i=r.return)&&i.call(r),0):r.next)&&!(i=i.call(r,u[1])).done)return i;switch(r=0,i&&(u=[2&u[0],i.value]),u[0]){case 0:case 1:i=u;break;case 4:return o.label++,{value:u[1],done:!1};case 5:o.label++,r=u[1],u=[0];continue;case 7:u=o.ops.pop(),o.trys.pop();continue;default:if(!(i=(i=o.trys).length>0&&i[i.length-1])&&(6===u[0]||2===u[0])){o=0;continue}if(3===u[0]&&(!i||u[1]>i[0]&&u[1]<i[3])){o.label=u[1];break}if(6===u[0]&&o.label<i[1]){o.label=i[1],i=u;break}if(i&&o.label<i[2]){o.label=i[2],o.ops.push(u);break}i[2]&&o.ops.pop(),o.trys.pop();continue}u=t.call(e,o)}catch(a){u=[6,a],r=0}finally{n=i=0}if(5&u[0])throw u[1];return{value:u[0]?u[1]:void 0,done:!0}}([u,a])}}}var o,a=function(){},s=a(),c=Object,f=function(e){return e===s},l=function(e){return"function"==typeof e},d=function(e,t){return c.assign({},e,t)},h="undefined",v=function(){return typeof window!=h},p=new WeakMap,y=0,g=function(e){var t,n,r=typeof e,i=e&&e.constructor,u=i==Date;if(c(e)!==e||u||i==RegExp)t=u?e.toJSON():"symbol"==r?e.toString():"string"==r?JSON.stringify(e):""+e;else{if(t=p.get(e))return t;if(t=++y+"~",p.set(e,t),i==Array){for(t="@",n=0;n<e.length;n++)t+=g(e[n])+",";p.set(e,t)}if(i==c){t="#";for(var o=c.keys(e).sort();!f(n=o.pop());)f(e[n])||(t+=n+":"+g(e[n])+",");p.set(e,t)}}return t},b=!0,m=v(),w=typeof document!=h,O=m&&window.addEventListener?window.addEventListener.bind(window):a,$=w?document.addEventListener.bind(document):a,_=m&&window.removeEventListener?window.removeEventListener.bind(window):a,S=w?document.removeEventListener.bind(document):a,M={isOnline:function(){return b},isVisible:function(){var e=w&&document.visibilityState;return f(e)||"hidden"!==e}},D={initFocus:function(e){return $("visibilitychange",e),O("focus",e),function(){S("visibilitychange",e),_("focus",e)}},initReconnect:function(e){var t=function(){b=!0,e()},n=function(){b=!1};return O("online",t),O("offline",n),function(){_("online",t),_("offline",n)}}},j=!v()||"Deno"in window,k=function(e){return v()&&typeof window.requestAnimationFrame!=h?window.requestAnimationFrame(e):setTimeout(e,1)},x=j?r.useEffect:r.useLayoutEffect,P="undefined"!==typeof navigator&&navigator.connection,E=!j&&P&&(["slow-2g","2g"].includes(P.effectiveType)||P.saveData),T=function(e){if(l(e))try{e=e()}catch(n){e=""}var t=[].concat(e);return[e="string"==typeof e?e:(Array.isArray(e)?e.length:e)?g(e):"",t,e?"$swr$"+e:""]},C=new WeakMap,R=function(e,t,n,r,i,u,o){void 0===o&&(o=!0);var a=C.get(e),s=a[0],c=a[1],f=a[3],l=s[t],d=c[t];if(o&&d)for(var h=0;h<d.length;++h)d[h](n,r,i);return u&&(delete f[t],l&&l[0])?l[0](2).then((function(){return e.get(t)})):e.get(t)},V=0,L=function(){return++V},I=function(){for(var e=[],t=0;t<arguments.length;t++)e[t]=arguments[t];return i(void 0,void 0,void 0,(function(){var t,n,r,i,o,a,c,h,v,p,y,g,b,m,w,O,$,_,S,M,D;return u(this,(function(u){switch(u.label){case 0:if(t=e[0],n=e[1],r=e[2],i=e[3],a=!!f((o="boolean"===typeof i?{revalidate:i}:i||{}).populateCache)||o.populateCache,c=!1!==o.revalidate,h=!1!==o.rollbackOnError,v=o.optimisticData,p=T(n),y=p[0],g=p[2],!y)return[2];if(b=C.get(t),m=b[2],e.length<3)return[2,R(t,y,t.get(y),s,s,c,!0)];if(w=r,$=L(),m[y]=[$,0],_=!f(v),S=t.get(y),_&&(M=l(v)?v(S):v,t.set(y,M),R(t,y,M)),l(w))try{w=w(t.get(y))}catch(j){O=j}return w&&l(w.then)?[4,w.catch((function(e){O=e}))]:[3,2];case 1:if(w=u.sent(),$!==m[y][0]){if(O)throw O;return[2,w]}O&&_&&h&&(a=!0,w=S,t.set(y,S)),u.label=2;case 2:return a&&(O||(l(a)&&(w=a(w,S)),t.set(y,w)),t.set(g,d(t.get(g),{error:O}))),m[y][1]=L(),[4,R(t,y,w,O,s,c,!!a)];case 3:if(D=u.sent(),O)throw O;return[2,a?D:w]}}))}))},N=function(e,t){for(var n in e)e[n][0]&&e[n][0](t)},W=function(e,t){if(!C.has(e)){var n=d(D,t),r={},i=I.bind(s,e),u=a;if(C.set(e,[r,{},{},{},i]),!j){var o=n.initFocus(setTimeout.bind(s,N.bind(s,r,0))),c=n.initReconnect(setTimeout.bind(s,N.bind(s,r,1)));u=function(){o&&o(),c&&c(),C.delete(e)}}return[e,i,u]}return[e,C.get(e)[4]]},A=W(new Map),Y=A[0],F=A[1],H=d({onLoadingSlow:a,onSuccess:a,onError:a,onErrorRetry:function(e,t,n,r,i){var u=n.errorRetryCount,o=i.retryCount,a=~~((Math.random()+.5)*(1<<(o<8?o:8)))*n.errorRetryInterval;!f(u)&&o>u||setTimeout(r,a,i)},onDiscarded:a,revalidateOnFocus:!0,revalidateOnReconnect:!0,revalidateIfStale:!0,shouldRetryOnError:!0,errorRetryInterval:E?1e4:5e3,focusThrottleInterval:5e3,dedupingInterval:2e3,loadingTimeout:E?5e3:3e3,compare:function(e,t){return g(e)==g(t)},isPaused:function(){return!1},cache:Y,mutate:F,fallback:{}},M),z=function(e,t){var n=d(e,t);if(t){var r=e.use,i=e.fallback,u=t.use,o=t.fallback;r&&u&&(n.use=r.concat(u)),i&&o&&(n.fallback=d(i,o))}return n},J=(0,r.createContext)({}),Z=function(e){return l(e[1])?[e[0],e[1],e[2]||{}]:[e[0],null,(null===e[1]?e[2]:e[1])||{}]},q=function(){return d(H,(0,r.useContext)(J))},U=function(e,t,n){var r=t[e]||(t[e]=[]);return r.push(n),function(){var e=r.indexOf(n);e>=0&&(r[e]=r[r.length-1],r.pop())}},G={dedupe:!0},B=c.defineProperty((function(e){var t=e.value,n=z((0,r.useContext)(J),t),i=t&&t.provider,u=(0,r.useState)((function(){return i?W(i(n.cache||Y),t):s}))[0];return u&&(n.cache=u[0],n.mutate=u[1]),x((function(){return u?u[2]:s}),[]),(0,r.createElement)(J.Provider,d(e,{value:n}))}),"default",{value:H}),X=(o=function(e,t,n){var o=n.cache,a=n.compare,c=n.fallbackData,h=n.suspense,v=n.revalidateOnMount,p=n.refreshInterval,y=n.refreshWhenHidden,g=n.refreshWhenOffline,b=C.get(o),m=b[0],w=b[1],O=b[2],$=b[3],_=T(e),S=_[0],M=_[1],D=_[2],P=(0,r.useRef)(!1),E=(0,r.useRef)(!1),V=(0,r.useRef)(S),N=(0,r.useRef)(t),W=(0,r.useRef)(n),A=function(){return W.current},Y=function(){return A().isVisible()&&A().isOnline()},F=function(e){return o.set(D,d(o.get(D),e))},H=o.get(S),z=f(c)?n.fallback[S]:c,J=f(H)?z:H,Z=o.get(D)||{},q=Z.error,B=!P.current,X=function(){return B&&!f(v)?v:!A().isPaused()&&(h?!f(J)&&n.revalidateIfStale:f(J)||n.revalidateIfStale)},Q=!(!S||!t)&&(!!Z.isValidating||B&&X()),K=function(e,t){var n=(0,r.useState)({})[1],i=(0,r.useRef)(e),u=(0,r.useRef)({data:!1,error:!1,isValidating:!1}),o=(0,r.useCallback)((function(e){var r=!1,o=i.current;for(var a in e){var s=a;o[s]!==e[s]&&(o[s]=e[s],u.current[s]&&(r=!0))}r&&!t.current&&n({})}),[]);return x((function(){i.current=e})),[i,u.current,o]}({data:J,error:q,isValidating:Q},E),ee=K[0],te=K[1],ne=K[2],re=(0,r.useCallback)((function(e){return i(void 0,void 0,void 0,(function(){var t,r,i,c,d,h,v,p,y,g,b,m,w;return u(this,(function(u){switch(u.label){case 0:if(t=N.current,!S||!t||E.current||A().isPaused())return[2,!1];c=!0,d=e||{},h=!$[S]||!d.dedupe,v=function(){return!E.current&&S===V.current&&P.current},p=function(){var e=$[S];e&&e[1]===i&&delete $[S]},y={isValidating:!1},g=function(){F({isValidating:!1}),v()&&ne(y)},F({isValidating:!0}),ne({isValidating:!0}),u.label=1;case 1:return u.trys.push([1,3,,4]),h&&(R(o,S,ee.current.data,ee.current.error,!0),n.loadingTimeout&&!o.get(S)&&setTimeout((function(){c&&v()&&A().onLoadingSlow(S,n)}),n.loadingTimeout),$[S]=[t.apply(void 0,M),L()]),w=$[S],r=w[0],i=w[1],[4,r];case 2:return r=u.sent(),h&&setTimeout(p,n.dedupingInterval),$[S]&&$[S][1]===i?(F({error:s}),y.error=s,b=O[S],!f(b)&&(i<=b[0]||i<=b[1]||0===b[1])?(g(),h&&v()&&A().onDiscarded(S),[2,!1]):(a(ee.current.data,r)?y.data=ee.current.data:y.data=r,a(o.get(S),r)||o.set(S,r),h&&v()&&A().onSuccess(r,S,n),[3,4])):(h&&v()&&A().onDiscarded(S),[2,!1]);case 3:return m=u.sent(),p(),A().isPaused()||(F({error:m}),y.error=m,h&&v()&&(A().onError(m,S,n),("boolean"===typeof n.shouldRetryOnError&&n.shouldRetryOnError||l(n.shouldRetryOnError)&&n.shouldRetryOnError(m))&&Y()&&A().onErrorRetry(m,S,n,re,{retryCount:(d.retryCount||0)+1,dedupe:!0}))),[3,4];case 4:return c=!1,g(),v()&&h&&R(o,S,y.data,y.error,!1),[2,!0]}}))}))}),[S]),ie=(0,r.useCallback)(I.bind(s,o,(function(){return V.current})),[]);if(x((function(){N.current=t,W.current=n})),x((function(){if(S){var e=S!==V.current,t=re.bind(s,G),n=0,r=U(S,w,(function(e,t,n){ne(d({error:t,isValidating:n},a(ee.current.data,e)?s:{data:e}))})),i=U(S,m,(function(e){if(0==e){var r=Date.now();A().revalidateOnFocus&&r>n&&Y()&&(n=r+A().focusThrottleInterval,t())}else if(1==e)A().revalidateOnReconnect&&Y()&&t();else if(2==e)return re()}));return E.current=!1,V.current=S,P.current=!0,e&&ne({data:J,error:q,isValidating:Q}),X()&&(f(J)||j?t():k(t)),function(){E.current=!0,r(),i()}}}),[S,re]),x((function(){var e;function t(){var t=l(p)?p(J):p;t&&-1!==e&&(e=setTimeout(n,t))}function n(){ee.current.error||!y&&!A().isVisible()||!g&&!A().isOnline()?t():re(G).then(t)}return t(),function(){e&&(clearTimeout(e),e=-1)}}),[p,y,g,re]),(0,r.useDebugValue)(J),h&&f(J)&&S)throw N.current=t,W.current=n,E.current=!1,f(q)?re(G):q;return{mutate:ie,get data(){return te.data=!0,J},get error(){return te.error=!0,q},get isValidating(){return te.isValidating=!0,Q}}},function(){for(var e=[],t=0;t<arguments.length;t++)e[t]=arguments[t];var n=q(),r=Z(e),i=r[0],u=r[1],a=r[2],s=z(n,a),c=o,f=s.use;if(f)for(var l=f.length;l-- >0;)c=f[l](c);return c(i,u||s.fetcher,s)})}},function(e){var t=function(t){return e(e.s=t)};e.O(0,[774,179],(function(){return t(1780),t(880)}));var n=e.O();_N_E=n}]);