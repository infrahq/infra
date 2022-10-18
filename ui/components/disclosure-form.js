import { Transition, Disclosure } from '@headlessui/react'
import { ChevronRightIcon } from '@heroicons/react/outline'

export default function DisclosureForm({ children, title }) {
  return (
    <Disclosure>
      {({ open }) => (
        <>
          <Disclosure.Button className='w-full'>
            <span className='flex items-center text-xs font-medium text-gray-500'>
              <ChevronRightIcon
                className={`${
                  open ? 'rotate-90 transform' : ''
                } mr-1 h-3 w-3 text-gray-500 duration-300 ease-in`}
              />
              {title}
            </span>
          </Disclosure.Button>
          <Transition
            show={open}
            enter='ease-out duration-1000'
            enterFrom='opacity-0'
            enterTo='opacity-100'
            leave='ease-in duration-300'
            leaveFrom='opacity-100'
            leaveTo='opacity-0'
          >
            <Disclosure.Panel static>{children}</Disclosure.Panel>
          </Transition>
        </>
      )}
    </Disclosure>
  )
}
