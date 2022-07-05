import * as React from 'react'

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
          className='relative hidden px-[0.2em] text-zinc-500 no-underline hover:underline group-hover:inline'
        >
          #
        </a>
      )}
    </Component>
  )
}
