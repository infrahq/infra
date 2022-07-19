import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/solid'

export default function Pagination(curr) {
  return (
    <nav className="pb-4 px-4 flex justify-end">  
        <a
          href="#"
          className=" py-2 pr-1 inline-flex items-center text-sm font-medium text-gray-500 hover:text-violet-300 "
        >
          <ChevronLeftIcon className=" h-5 w-5 text-gray-400 hover:text-violet-300" aria-hidden="true" />
          
        </a>
        <a
          href="#"
          className="text-gray-500 hover:text-violet-300 py-2 px-4 inline-flex items-center text-sm font-medium"
        >
          1
        </a>
        {/* Current: "border-indigo-500 text-indigo-600", Default: "text-gray-500 hover:text-violet-300 " */}
        <a
          href="#"
          className="text-white hover:text-violet-300 py-2 px-4 inline-flex items-center text-sm font-medium rounded-md bg-gray-700"
          aria-current="page"
        >
          2
        </a>
        <a
          href="#"
          className="text-gray-500 hover:text-violet-300  py-2 px-4 inline-flex items-center text-sm font-medium"
        >
          3
        </a>
        <a
          href="#"
          className="text-gray-500 hover:text-violet-300   py-2 px-4 inline-flex items-center text-sm font-medium"
        >
          4
        </a>
        <a
          href="#"
          className="text-gray-500 hover:text-violet-300   py-2 px-4 inline-flex items-center text-sm font-medium"
        >
          5
        </a>
        <a
          href="#"
          className="text-gray-500 hover:text-violet-300   py-2 px-4 inline-flex items-center text-sm font-medium"
        >
          6
        </a>
        <a
          href="#"
          className="py-2 pl-1 inline-flex items-center text-sm font-medium text-gray-500 hover:text-violet-300 "
        >
          <ChevronRightIcon className="ml-3 h-5 w-5 text-gray-400 hover:text-violet-300 " aria-hidden="true" />
        </a>
    </nav>
  )
}
