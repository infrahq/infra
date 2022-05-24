(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[616],{9139:function(e,r,t){(window.__NEXT_P=window.__NEXT_P||[]).push(["/signup",function(){return t(3006)}])},5686:function(e,r,t){"use strict";t.d(r,{Z:function(){return s}});var n=t(5893);function s(e){var r=e.message,t=e.center,s=void 0!==t&&t;return(0,n.jsx)("p",{className:"".concat(s?"mt-2 text-center":"mb-1"," text-xs text-pink-500"),children:r})}},5876:function(e,r,t){"use strict";t.d(r,{Z:function(){return s}});var n=t(5893);function s(e){var r=e.children;return(0,n.jsx)("div",{className:"w-full min-h-full flex flex-col justify-center",children:(0,n.jsxs)("div",{className:"flex flex-col w-full max-w-xs mx-auto justify-center items-center my-8 px-5 pt-8 pb-4 border rounded-lg border-gray-800",children:[(0,n.jsx)("div",{className:"border border-violet-200/25 rounded-full p-2.5 mb-4",children:(0,n.jsx)("img",{className:"w-12 h-12",src:"/infra-color.svg"})}),r]})})}},3006:function(e,r,t){"use strict";t.r(r),t.d(r,{default:function(){return x}});var n=t(4051),s=t.n(n),a=t(5893),o=t(7294),i=t(8100),l=t(1163),c=t(5876),u=t(5686);function d(e,r,t,n,s,a,o){try{var i=e[a](o),l=i.value}catch(c){return void t(c)}i.done?r(l):Promise.resolve(l).then(n,s)}function f(e){return function(){var r=this,t=arguments;return new Promise((function(n,s){var a=e.apply(r,t);function o(e){d(a,n,s,o,i,"next",e)}function i(e){d(a,n,s,o,i,"throw",e)}o(void 0)}))}}function x(){var e=(0,i.kY)().mutate,r=(0,l.useRouter)(),t=(0,o.useState)(""),n=t[0],c=t[1],d=(0,o.useState)(""),x=d[0],p=d[1],m=(0,o.useState)(""),b=m[0],h=m[1],v=(0,o.useState)({}),w=v[0],y=v[1];function g(){return(g=f(s().mark((function t(a){var o,i,l,c,u,d,f,p;return s().wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return a.preventDefault(),y({}),h(""),t.prev=3,t.next=6,fetch("/v1/signup",{method:"POST",body:JSON.stringify({email:n,password:x})});case 6:if((o=t.sent).ok){t.next=11;break}return t.next=10,o.json();case 10:throw t.sent;case 11:return t.next=13,fetch("/v1/login",{method:"POST",body:JSON.stringify({passwordCredentials:{email:n,password:x}})});case 13:if((o=t.sent).ok){t.next=18;break}return t.next=17,o.json();case 17:throw t.sent;case 18:e("/v1/identities/self",{optimisticData:{name:n}}),e("/v1/signup",{optimisticData:{enabled:!1}}),r.replace("/"),t.next=48;break;case 23:if(t.prev=23,t.t0=t.catch(3),!t.t0.fieldErrors){t.next=47;break}for(i={},l=!0,c=!1,u=void 0,t.prev=28,d=t.t0.fieldErrors[Symbol.iterator]();!(l=(f=d.next()).done);l=!0)p=f.value,i[p.fieldName.toLowerCase()]=p.errors[0]||"invalid value";t.next=36;break;case 32:t.prev=32,t.t1=t.catch(28),c=!0,u=t.t1;case 36:t.prev=36,t.prev=37,l||null==d.return||d.return();case 39:if(t.prev=39,!c){t.next=42;break}throw u;case 42:return t.finish(39);case 43:return t.finish(36);case 44:y(i),t.next=48;break;case 47:h(t.t0.message);case 48:return t.abrupt("return",!1);case 49:case"end":return t.stop()}}),t,null,[[3,23],[28,32,36,44],[37,,39,43]])})))).apply(this,arguments)}return(0,a.jsxs)(a.Fragment,{children:[(0,a.jsx)("h1",{className:"text-base leading-snug font-bold",children:"Welcome to Infra"}),(0,a.jsxs)("h2",{className:"text-xs text-center max-w-md my-1.5 text-gray-400",children:["You've successfully installed Infra.",(0,a.jsx)("br",{}),"Set up your admin user to get started."]}),(0,a.jsxs)("form",{onSubmit:function(e){return g.apply(this,arguments)},className:"flex flex-col w-full max-w-sm",children:[(0,a.jsxs)("div",{className:"w-full my-4",children:[(0,a.jsx)("label",{htmlFor:"email",className:"text-3xs text-gray-400 uppercase",children:"Email"}),(0,a.jsx)("input",{autoFocus:!0,name:"email",type:"email",placeholder:"email@address.com",onChange:function(e){return c(e.target.value)},className:"w-full bg-transparent border-b border-gray-800 text-2xs px-px mt-2 py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ".concat(w.email?"border-pink-500/60":"")}),w.email&&(0,a.jsx)(u.Z,{message:w.email})]}),(0,a.jsxs)("div",{className:"w-full my-4",children:[(0,a.jsx)("label",{htmlFor:"password",className:"text-3xs text-gray-400 uppercase",children:"Password"}),(0,a.jsx)("input",{type:"password",placeholder:"enter your password",onChange:function(e){return p(e.target.value)},className:"w-full bg-transparent border-b border-gray-800 text-2xs px-px mt-2 py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ".concat(w.password?"border-pink-500/60":"")}),w.password&&(0,a.jsx)(u.Z,{message:w.password})]}),(0,a.jsxs)("button",{disabled:!n||!x,className:"border border-violet-300 hover:border-violet-100 my-2 text-2xs px-4 py-3 rounded-lg disabled:pointer-events-none text-violet-100 disabled:opacity-30",children:["Get Started",b&&(0,a.jsx)(u.Z,{message:b,center:!0})]})]})]})}x.layout=function(e){return(0,a.jsx)(c.Z,{children:e})}}},function(e){e.O(0,[774,888,179],(function(){return r=9139,e(e.s=r);var r}));var r=e.O();_N_E=r}]);