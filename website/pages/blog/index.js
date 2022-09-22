import React from 'react'
import Head from 'next/head'
import dayjs from 'dayjs'
import Link from 'next/link'
import Markdoc from '@markdoc/markdoc'

import components from '../../lib/markdoc/components'
import { posts } from '../../lib/blog'
import Layout from '../../components/layout'

export default function Blog({ posts }) {
  return (
    <>
      <Head>
        <title>Infra - Blog</title>
        <meta property='og:title' content='Infra - Blog' key='title' />
        <meta property='og:url' content='https://infrahq.com/blog' />
        <meta property='og:description' content='Infra Blog' />
      </Head>
      <section className='mx-auto w-full max-w-3xl p-6'>
        <div className='my-16'>
          <h1 className='my-2 font-display text-3xl font-bold md:my-6 md:text-5xl'>
            Infra Blog
          </h1>
          <h2 className='mt-2 mb-4 font-display text-xl text-gray-600'>
            The latest product updates and news from Infra
          </h2>
          <a
            className='font-medium text-blue-500'
            target='_blank'
            href='https://twitter.com/infrahq'
            rel='noreferrer'
          >
            Follow Infra on Twitter ›
          </a>
        </div>
        {posts.map(p => (
          <div key={p.title}>
            <h1 className='mb-4 font-display text-4xl font-semibold'>
              <Link href={p.href}>
                <a>{p.title}</a>
              </Link>
            </h1>
            <h2 className='flex items-baseline text-sm font-semibold text-gray-500'>
              {p.date && dayjs(p.date).format('MMMM D, YYYY')}
              {p.date && p.author && <span className='px-2'>·</span>}
              {p.author}
            </h2>
            <div className='prose-docs prose-md prose w-full max-w-none break-words'>
              {Markdoc.renderers.react(JSON.parse(p.markdoc), React, {
                components,
              })}
            </div>
            <hr className='my-16' />
          </div>
        ))}
      </section>
    </>
  )
}

Blog.layout = page => {
  return <Layout>{page}</Layout>
}

export async function getStaticProps() {
  return {
    props: {
      posts: posts(),
    },
  }
}
