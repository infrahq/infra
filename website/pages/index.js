import Head from 'next/head'
import Link from 'next/link'

import {
  StarIcon,
  IdentificationIcon,
  ArrowPathIcon,
  BoltIcon,
  ShieldCheckIcon,
} from '@heroicons/react/24/outline'
import Layout from '../components/layout'

export default function Index() {
  return (
    <>
      <Head>
        <title>Infra - Infrastructure Access</title>
        <meta
          property='og:title'
          content='Single sign-on for your infrastructure'
          key='title'
        />
        <meta
          property='og:description'
          content='Infra enables single sign-on for your infrastructure'
        />
      </Head>
      <section className='flex flex-1 flex-col px-4'>
        <div className='relative mx-auto my-24 flex w-full max-w-7xl flex-1 items-center justify-between space-x-16'>
          <div className='flex max-w-2xl flex-1 flex-col'>
            <h1 className='my-4 overflow-visible text-3xl font-bold tracking-tight md:text-6xl'>
              Connect your team to your infrastructure.
            </h1>
            <h2 className='text-md max-w-xl text-gray-600 md:text-2xl'>
              Infra is the easiest way to manage access to Kubernetes, with more
              connectors coming soon.
            </h2>
            <div className='z-40 my-5 flex items-center space-x-2 text-base'>
              <Link href='/docs/getting-started/quickstart'>
                <a className='rounded-full bg-blue-500 py-1.5 px-4 text-base font-semibold text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 md:py-2 md:px-5 md:text-lg'>
                  Get Started
                </a>
              </Link>
              <Link href='https://github.com/infrahq/infra'>
                <a className='flex items-center rounded-full border border-gray-300 py-1.5 px-3 text-base font-semibold text-gray-500 hover:border-gray-400 hover:text-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 md:py-2 md:px-4 md:text-lg'>
                  <StarIcon className='relative mr-1.5 h-5 stroke-current' />{' '}
                  Star on GitHub
                </a>
              </Link>
            </div>
          </div>
        </div>
      </section>
      <section className='mb-12 flex flex-1 flex-col px-4'>
        <div className='mx-auto grid w-full max-w-7xl grid-cols-2 gap-12 md:grid-cols-4'>
          <div>
            <BoltIcon className='h-7 stroke-1 text-gray-700' />
            <h3 className='my-1.5 text-sm font-medium'>
              Discover &amp; access in one place
            </h3>
            <p className='text-sm text-gray-500'>
              Share and discover access in minutes. Infra makes connecting to
              infrastructure in a single place fast and easy.
            </p>
          </div>
          <div>
            <ArrowPathIcon className='h-7 stroke-1 text-gray-700' />
            <h3 className='my-1.5 text-sm font-medium'>
              Keep credentials up to date
            </h3>
            <p className='text-sm text-gray-500'>
              No more out-of-date configurations or credentials. Infra provides
              short-lived credentials on the fly to all team members.
            </p>
          </div>
          <div>
            <IdentificationIcon className='h-7 stroke-1 text-gray-700' />
            <h3 className='my-1.5 text-sm font-medium'>
              Identity Provider Support
            </h3>
            <p className='text-sm text-gray-500'>
              Automatically on-board and off-board users via existing identity
              providers such as Google, Okta &amp; more.
            </p>
          </div>
          <div>
            <ShieldCheckIcon className='h-7 stroke-1 text-gray-700' />
            <h3 className='my-1.5 text-sm font-medium'>
              Secure, least-privilege access
            </h3>
            <p className='text-sm text-gray-500'>
              Provide short-lived and fine-grained access to specific
              infrastructure resources.
            </p>
          </div>
        </div>
      </section>
    </>
  )
}

Index.layout = page => <Layout>{page}</Layout>
