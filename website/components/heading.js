import * as React from 'react'
import { LinkIcon } from '@heroicons/react/24/solid'

export default function Heading({
  id = '',
  level = 1,
  children,
  className = '',
}) {
  const Component = `h${level}`
  const showAnchor = level > 1 && id

  return (
    <Component className={`${className} group relative scroll-mt-20`} id={id}>
      {children}
      {showAnchor && (
        <a
          href={`#${id}`}
          className='relative hidden px-[0.2em] no-underline hover:underline group-hover:inline'
        >
          <LinkIcon className='inline h-5 p-0.5 text-gray-400' />
        </a>
      )}
    </Component>
  )
}
