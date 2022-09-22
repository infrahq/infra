import path from 'path'
import React from 'react'
import Head from 'next/head'
import Link from 'next/link'
import dayjs from 'dayjs'
import Markdoc from '@markdoc/markdoc'
import { ArrowLeftIcon } from '@heroicons/react/24/solid'

import SignupForm from '../../components/signup-form'
import { posts } from '../../lib/blog'
import Layout from '../../components/layout'
import components from '../../lib/markdoc/components'

export default function Blog({ markdoc, date, author, title }) {
  return (
    <>
      <Head>
        <title>{`${title} - Infra Blog`}</title>
        <meta property='og:title' content={title} key='title' />
        <meta property='og:url' content='https://infrahq.com' />
        <meta property='og:description' content={`${title} - Infra Blog`} />
      </Head>
      <section className='mx-auto my-6 w-full max-w-3xl p-6 md:my-20'>
        <Link href='/blog'>
          <a className='mb-10 flex items-baseline font-medium text-blue-500'>
            <ArrowLeftIcon className='mr-2 h-3' /> Infra Blog
          </a>
        </Link>
        <div key={title}>
          <h1 className='mb-3 font-display text-5xl font-semibold'>{title}</h1>
          <h2 className='flex items-baseline text-sm font-semibold text-gray-500'>
            {date && dayjs(date).format('MMMM D, YYYY')}
            {date && author && <span className='px-2'>·</span>}
            {author}
          </h2>
          <div className='prose-docs prose-md prose w-full max-w-none break-words'>
            {Markdoc.renderers.react(JSON.parse(markdoc), React, {
              components,
            })}
          </div>
          <hr className='my-16' />
          <div className='my-8 flex flex-col items-center'>
            <h3 className='mb-6 text-xl font-bold tracking-tight'>
              Sign up for updates
            </h3>
            <SignupForm />
          </div>
        </div>
      </section>
    </>
  )
}

Blog.layout = page => {
  return <Layout>{page}</Layout>
}

export async function getStaticProps({ params }) {
  const post = posts().filter(p => path.basename(p.href) === params.slug)?.[0]
  return { props: { ...post } }
}

export async function getStaticPaths() {
  return {
    paths: posts().map(p => ({ params: { slug: path.basename(p.href) } })),
    fallback: false,
  }
}
