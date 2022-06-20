import * as React from 'react'

export default function Heading ({ id = '', level = 1, children, className = '' }) {
  const Component = `h${level}`
  const showAnchor = level > 1 && id

  return (
    <Component className={`${className} group scroll-mt-20 relative`} id={id}>
      {children}{showAnchor && <a href={`#${id}`} className='hidden group-hover:inline text-zinc-500 relative px-[0.2em] no-underline hover:underline'>#</a>}
    </Component>
  )
}
