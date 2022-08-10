import { InformationCircleIcon } from '@heroicons/react/outline'

export default function Tooltip({ children }) {
  return (
    <div class='group relative flex'>
      <InformationCircleIcon className='ml-0.5 inline h-4 w-4' />
      <div class='absolute bottom-0 mb-6 hidden flex-col group-hover:flex'>
        <span class='whitespace-no-wrap relative z-10 block w-[20rem] bg-black p-2 text-xs leading-none text-white shadow-lg'>
          {children}
        </span>
      </div>
    </div>
  )
}
