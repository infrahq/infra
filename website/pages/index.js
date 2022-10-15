import Head from 'next/head'
import Link from 'next/link'
import {
  StarIcon,
  IdentificationIcon,
  ArrowPathIcon,
  CommandLineIcon,
  ShieldCheckIcon,
} from '@heroicons/react/24/outline'

import Layout from '../components/layout'

const table = `
NAME   READY   STATUS
web    3/3     Running
api    1/1     Running
db     1/1     Running
cache  1/1     Running
`

export default function Index() {
  return (
    <>
      <Head>
        <title>Infra - Infrastructure Access</title>
        <meta property='og:title' content='Infrastructure Access' key='title' />
        <meta
          property='og:description'
          content='Connect your team to your Kubernetes'
        />
      </Head>
      <section className='flex flex-1 flex-col px-6'>
        <div className='relative mx-auto my-24 flex w-full max-w-7xl flex-1 flex-col items-center justify-between md:flex-row md:space-x-16'>
          <div className='flex flex-1 flex-col font-display'>
            <h1 className='my-4 overflow-visible text-3xl font-bold md:text-4xl lg:text-6xl'>
              Connect your team to your Kubernetes.
            </h1>
            <h2 className='text-md max-w-xl font-[450] text-gray-600 md:text-lg lg:text-2xl'>
              Infra is the easiest way to manage secure access to Kubernetes,
              with more integrations coming soon.
            </h2>
            <div className='space-x-0font-display z-40 my-6 flex flex-col space-y-4 text-base lg:flex-row lg:items-center lg:space-x-2 lg:space-y-0'>
              <Link href='/signup'>
                <a className='group relative rounded-full bg-black  py-1.5 px-4 text-center font-semibold text-white transition-colors hover:bg-zinc-800 focus:outline-none focus:ring-2 focus:ring-black focus:ring-offset-2 md:py-2 md:px-5 md:text-lg'>
                  Sign up for early access{' '}
                  <span className='inline-block transition-transform group-hover:translate-x-0.5'>
                    ›
                  </span>
                </a>
              </Link>
              <Link href='https://github.com/infrahq/infra'>
                <a className='flex items-center justify-center rounded-full border border-gray-300 py-1.5 px-3 text-base font-semibold text-gray-500 transition-colors hover:border-gray-400 hover:text-gray-600 focus:outline-none focus:ring-2 focus:ring-black focus:ring-offset-2 md:py-2 md:px-4 md:text-lg'>
                  <StarIcon className='relative mr-1.5 h-5 stroke-current' />{' '}
                  Star on GitHub
                </a>
              </Link>
            </div>
          </div>
          <div className='my-12 w-full max-w-xl flex-1 rounded-3xl bg-black px-8 py-8 font-mono leading-relaxed text-zinc-200 shadow-2xl shadow-black/40'>
            <div className='mb-1 font-semibold text-white'>
              $ kubectl get pods
            </div>
            <div>
              ? Select a login method:{' '}
              <span className='text-yellow-400'>Google</span>
            </div>
            <div>
              Logging in with{' '}
              <span className='font-semibold text-white'>Google...</span>
            </div>
            <div>
              Logged in as{' '}
              <span className='font-semibold text-white'>suzie@acme.com</span>
            </div>
            <div>
              {table
                .replace(/ /g, '\u00A0')
                .split('\n')
                .map(r => (
                  <div key={r}>{r}</div>
                ))}
            </div>
          </div>
        </div>
      </section>
      <section className='mb-12 flex flex-1 flex-col px-6'>
        <div className='mx-auto grid w-full max-w-7xl grid-cols-2 gap-12 md:grid-cols-4'>
          <div>
            <CommandLineIcon className='h-7 stroke-1 text-gray-700' />
            <h3 className='my-1.5 text-sm font-medium'>
              Access that just works
            </h3>
            <p className='text-sm text-gray-500'>
              Discover and access infrastructure hosted anywhere, in a single
              place.
            </p>
          </div>
          <div>
            <ArrowPathIcon className='h-7 stroke-1 text-gray-700' />
            <h3 className='my-1.5 text-sm font-medium'>
              Keep credentials up to date
            </h3>
            <p className='text-sm text-gray-500'>
              No more out-of-date configurations. Infra automatically refreshes
              credentials.
            </p>
          </div>
          <div>
            <IdentificationIcon className='h-7 stroke-1 text-gray-700' />
            <h3 className='my-1.5 text-sm font-medium'>
              Import users &amp; groups
            </h3>
            <p className='text-sm text-gray-500'>
              On-board and off-board users via identity providers such as Okta
              and Google.
            </p>
          </div>
          <div>
            <ShieldCheckIcon className='h-7 stroke-1 text-gray-700' />
            <h3 className='my-1.5 text-sm font-medium'>
              Secure &amp; easy to use
            </h3>
            <p className='text-sm text-gray-500'>
              Deploy in minutes. Provide fine-grained access to individual
              resources.
            </p>
          </div>
        </div>
      </section>
    </>
  )
}

Index.layout = page => <Layout>{page}</Layout>
