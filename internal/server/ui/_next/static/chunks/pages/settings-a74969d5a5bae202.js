(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[662],{3733:function(e,n,t){(window.__NEXT_P=window.__NEXT_P||[]).push(["/settings",function(){return t(6483)}])},5686:function(e,n,t){"use strict";t.d(n,{Z:function(){return i}});var r=t(5893);function i(e){var n=e.message,t=e.center,i=void 0!==t&&t;return(0,r.jsx)("p",{className:"".concat(i?"mt-2 text-center":"px-4 mb-1"," text-sm text-pink-500"),children:n})}},3783:function(e,n,t){"use strict";t.d(n,{Z:function(){return i}});var r=t(5893);function i(e){var n=e.label,t=e.type,i=e.value,a=e.placeholder,s=e.error,c=e.hasDropdownSelection,l=void 0===c||c,o=e.optionType,u=e.options,d=e.handleInputChange,f=e.handleSelectOption,m=e.handleKeyDown;return(0,r.jsxs)("div",{children:[n&&(0,r.jsx)("label",{htmlFor:"price",className:"block text-sm font-medium text-white",children:n}),(0,r.jsxs)("div",{className:"relative rounded shadow-sm",children:[(0,r.jsx)("input",{type:t,value:i,className:"block w-full px-4 py-3 sm:text-sm border-2 bg-transparent rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ".concat(s?"border-pink-500":"border-gray-800"),placeholder:a,onChange:d,onKeyDown:m}),l&&(0,r.jsxs)("div",{className:"absolute inset-y-0 right-2 flex items-center",children:[(0,r.jsx)("label",{htmlFor:o,className:"sr-only",children:o}),(0,r.jsx)("select",{id:o,name:o,onChange:f,className:"h-full py-0 pl-2 border-transparent bg-transparent text-white text-sm focus:outline-none",children:u.map((function(e){return(0,r.jsx)("option",{value:e,children:e},e)}))})]})]})]})}},8324:function(e,n,t){"use strict";t.d(n,{o:function(){return r}});var r=function(e){return String(e).toLowerCase().match(/^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/)}},4584:function(e,n,t){"use strict";t.r(n),t.d(n,{default:function(){return g}});var r=t(5893),i=t(7294),a=t(8100),s=t(9521),c=t(8324),l=t(3783),o=t(5857),u=t(1399),d=t(5686);function f(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function m(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{},r=Object.keys(t);"function"===typeof Object.getOwnPropertySymbols&&(r=r.concat(Object.getOwnPropertySymbols(t).filter((function(e){return Object.getOwnPropertyDescriptor(t,e).enumerable})))),r.forEach((function(n){f(e,n,t[n])}))}return e}var h=[{id:"name",accessor:function(e){return e},Cell:function(e){var n=e.value;return(0,r.jsx)(x,{id:n.subject})}},{id:"delete",accessor:function(e){return e},Cell:function(e){var n=e.value,t=(0,a.ZP)("/v1/identities/".concat(n.subject.replace("i:","")),{fallbackData:{name:"",kind:""}}).data,s=(0,a.kY)().mutate,c=(0,i.useState)(!1),l=c[0],o=c[1];return(0,r.jsxs)("div",{className:"opacity-0 group-hover:opacity-100 flex justify-end text-right",children:[(0,r.jsx)("button",{onClick:function(){return o(!0)},className:"p-2 -mr-2 cursor-pointer text-gray-500 hover:text-white",children:"Revoke"}),(0,r.jsx)(u.Z,{open:l,setOpen:o,onSubmit:function(){fetch("/v1/grants/".concat(n.id),{method:"DELETE"}).then((function(){return o(!1)})).finally((function(){return s("/v1/grants?resource=infra&privilege=admin")})).catch((function(e){console.error(e)}))},title:"Delete Admin",message:(0,r.jsxs)(r.Fragment,{children:["Are you sure you want to delete ",(0,r.jsx)("span",{className:"font-bold text-white",children:t.name}),"? This action cannot be undone."]})})]})}}],x=function(e){var n=e.id,t=(0,a.ZP)("/v1/identities/".concat(n.replace("i:","")),{fallbackData:{name:"",kind:""}}).data;return(0,r.jsxs)("div",{className:"flex items-center",children:[(0,r.jsx)("div",{className:"w-10 h-10 mr-4 bg-purple-100/10 font-bold rounded-lg flex items-center justify-center",children:t.name&&t.name[0].toUpperCase()}),(0,r.jsxs)("div",{className:"flex flex-col leading-tight",children:[(0,r.jsx)("div",{className:"font-medium",children:t.name}),(0,r.jsx)("div",{className:"text-gray-400 text-xs",children:t.kind})]})]})};function g(){var e=(0,a.ZP)((function(){return"/v1/grants?resource=infra&privilege=admin"}),{fallbackData:[]}).data,n=(0,a.kY)().mutate,t=(0,s.useTable)({columns:h,data:e||[]}),u=(0,i.useState)(""),f=u[0],x=u[1],g=(0,i.useState)(""),v=g[0],p=g[1],b=function(e){fetch("/v1/grants",{method:"POST",body:JSON.stringify({subject:"i:"+e,resource:"infra",privilege:"admin"})}).then((function(){n("/v1/grants?resource=infra&privilege=admin"),x("")})).catch((function(e){return p(e.message||"something went wrong, please try again later.")}))},j=function(){(0,c.o)(f)?(p(""),fetch("/v1/identities?name=".concat(f)).then((function(e){return e.json()})).then((function(e){0===e.length?fetch("/v1/identities",{method:"POST",body:JSON.stringify({name:f,kind:"user"})}).then((function(e){return e.json()})).then((function(e){return b(e.id)})).catch((function(e){return console.error(e)})):b(e[0].id)}))):p("Invalid email")};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)("h3",{className:"text-lg font-bold mb-4",children:"Admins"}),(0,r.jsx)("h4",{className:"text-gray-300 mb-4 text-sm w-3/4",children:"Infra admins have full access to the Infra API, including creating additional grants, managing identity providers, managing destinations, and managing other users."}),(0,r.jsxs)("div",{className:"flex gap-1 ".concat(v?"mt-10 mb-2":"my-10"," my-10 w-3/4"),children:[(0,r.jsx)("div",{className:"flex-1 w-full",children:(0,r.jsx)(l.Z,{type:"email",value:f,placeholder:"email",hasDropdownSelection:!1,handleInputChange:function(e){return n=e.target.value,x(n),void p("");var n},handleKeyDown:function(e){"Enter"===e.key&&f.length>0&&j()},error:v})}),(0,r.jsx)("button",{onSubmit:function(){return j()},disabled:0===f.length,type:"submit",className:"bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 p-0.5 mx-auto rounded-full",children:(0,r.jsx)("div",{className:"bg-black flex items-center text-sm px-14 py-3 rounded-full",children:"Add"})})]}),v&&(0,r.jsx)(d.Z,{message:v}),(0,r.jsx)("h4",{className:"text-gray-400 my-3 text-sm",children:"These  users have full administration privileges"}),e&&e.length>0&&(0,r.jsx)("div",{className:"w-3/4",children:(0,r.jsx)(o.Z,m({},t,{showHeader:!1}))})]})}},6483:function(e,n,t){"use strict";t.r(n),t.d(n,{default:function(){return l}});var r=t(5893),i=t(9008),a=t(9540),s=t(1431),c=t(4584);function l(){return(0,r.jsxs)("div",{className:"flex flex-row mt-4 mb-4 lg:mt-6",children:[(0,r.jsx)(i.default,{children:(0,r.jsx)("title",{children:"Settings - Infra"})}),(0,r.jsx)("div",{className:"mt-2 mr-6",children:(0,r.jsx)(s.Z,{iconPath:"/settings-color.svg"})}),(0,r.jsxs)("div",{className:"flex-1 flex flex-col space-y-4",children:[(0,r.jsx)("h1",{className:"text-2xl font-bold mt-6 mb-4",children:"Settings"}),(0,r.jsx)("div",{className:"pt-3",children:(0,r.jsx)(c.default,{})})]})]})}l.layout=function(e){return(0,r.jsx)(a.Z,{children:e})}}},function(e){e.O(0,[642,203,774,888,179],(function(){return n=3733,e(e.s=n);var n}));var n=e.O();_N_E=n}]);