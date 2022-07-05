import Head from 'next/head'
import Link from 'next/link'

import SignupForm from '../components/signup-form'
import Layout from '../components/layout'

export default function Index() {
  return (
    <>
      <Head>
        <title>Infra - Single sign-on for your infrastructure</title>
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
      <section className='relative mx-auto mb-48 flex w-full max-w-screen-2xl flex-1 flex-col items-center px-8 lg:mb-12 lg:flex-row'>
        <div className='flex-1 text-center lg:text-left'>
          <h1 className='my-12 max-w-lg text-center text-6xl font-light tracking-tighter lg:my-4 lg:max-w-none lg:text-left xl:text-[72px]'>
            The simplest way to manage infrastructure access
          </h1>
          <ul className='lg:mx-none my-16 mx-auto max-w-md space-y-1 text-lg leading-tight tracking-tight text-gray-300 md:leading-tight lg:my-4 lg:mx-0 lg:text-xl lg:leading-tight xl:max-w-xl'>
            <li className='flex items-start'>
              <img
                alt='check'
                src='/images/check.svg'
                className='mr-3 mt-1 h-5 w-5'
              />
              <span className='text-left'>
                In one command, login and discover access
              </span>
            </li>
            <li className='flex items-start'>
              <img
                alt='check'
                src='/images/check.svg'
                className='mr-3 mt-1 h-5 w-5'
              />
              <span className='text-left'>
                Short-lived, auto-refreshed credentials under the hood
              </span>
            </li>
            <li className='flex items-start'>
              <img
                alt='check'
                src='/images/check.svg'
                className='mr-3 mt-1 h-5 w-5'
              />
              <span className='text-left'>
                Nothing else to set up for your team
              </span>
            </li>
            <li className='flex items-start pt-3'>
              <img
                alt='kubernetes icon'
                src='/images/kubernetes.svg'
                className='mr-3 h-[21px] w-[21px]'
              />{' '}
              <span className='font-bold text-white'>
                Available for Kubernetes
              </span>
            </li>
          </ul>
          <div className='xl:mx-none relative z-40 my-16 mx-auto flex max-w-sm flex-1 flex-col items-stretch justify-center space-y-6 lg:my-10 lg:mx-0 lg:max-w-md xl:max-w-xl xl:flex-row xl:items-start xl:justify-start xl:space-y-0 xl:space-x-4'>
            <Link href='https://github.com/infrahq/infra'>
              <div className='group inline-flex flex-none overflow-hidden rounded-full bg-gradient-to-tr from-cyan-100 to-pink-300'>
                <button className='m-0.5 flex w-full justify-center rounded-full bg-black py-2 pr-5 pl-3 text-lg text-gray-100 group-hover:bg-gray-900 group-hover:text-white'>
                  <img
                    alt='github logo'
                    className='pr-3'
                    src='/images/github.svg'
                  />{' '}
                  Open in GitHub
                </button>
              </div>
            </Link>
            <div className='flex flex-1 md:mr-0'>
              <SignupForm />
            </div>
          </div>
        </div>
        <div className='relative flex max-w-5xl flex-1 items-center justify-center'>
          <img
            alt='hero image'
            className='relative top-24 block scale-125 md:top-16 md:scale-100 lg:-right-12 lg:scale-125'
            src='https://user-images.githubusercontent.com/251292/174117176-680ee285-dd39-4a0b-a0b8-b75a09ccdc2e.png'
          />
        </div>
      </section>
    </>
  )
}

Index.layout = page => <Layout>{page}</Layout>
