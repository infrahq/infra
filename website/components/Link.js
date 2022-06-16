import NextLink from 'next/link'

export default function ({ href, children }) {
  const target = href.startsWith('http') ? '_blank' : undefined

  return (
    <NextLink href={href} passHref>
      <a target={target} rel={target === '_blank' ? 'noreferrer' : undefined}>
        {children}
      </a>
    </NextLink>
  )
}
