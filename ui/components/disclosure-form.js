import { Transition, Disclosure } from '@headlessui/react'
import { ChevronRightIcon } from '@heroicons/react/outline'

export default function DisclosureForm({
  children,
  title,
  defaultOpen = false,
}) {
  return (
    <Disclosure defaultOpen={defaultOpen}>
      {({ open }) => (
        <>
          <Disclosure.Button className='w-full'>
            <span className='flex items-center text-xs font-medium text-gray-500'>
              <ChevronRightIcon
                className={`${
                  open ? 'rotate-90 transform' : ''
                } mr-1 h-3 w-3 text-gray-500`}
              />
              {title}
            </span>
          </Disclosure.Button>
          <Transition show={open}>
            <Disclosure.Panel static>{children}</Disclosure.Panel>
          </Transition>
        </>
      )}
    </Disclosure>
  )
}
