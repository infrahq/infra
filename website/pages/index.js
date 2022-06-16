import Head from 'next/head'
import Link from 'next/link'
import SignupForm from '../components/SignupForm'

import Layout from '../components/Layout'

export default function Index () {
  return (
    <>
      <Head>
        <title>Infra - Single sign-on for your infrastructure</title>
        <meta property='og:title' content='Single sign-on for your infrastructure' key='title' />
        <meta property='og:description' content='Infra enables single sign-on for your infrastructure' />
      </Head>
      <section className='flex flex-col flex-1 mb-48 lg:mb-12 lg:flex-row items-center relative max-w-screen-2xl px-8 w-full mx-auto'>
        <div className='flex-1 text-center lg:text-left'>
          <h1 className='my-12 lg:my-4 text-center max-w-lg lg:max-w-none lg:text-left text-6xl xl:text-[72px] 2xl:leading-[0.9] font-light tracking-tighter'>
            The simplest way to manage infrastructure access
          </h1>
          <ul className='space-y-1 my-16 lg:my-4 mx-auto lg:mx-none lg:mx-0 max-w-md xl:max-w-xl text-lg lg:text-xl tracking-tight leading-tight md:leading-tight lg:leading-tight text-gray-300'>
            <li className='flex items-start'><img src='/images/check.svg' className='w-5 h-5 mr-3 mt-1' /><span className='text-left'>In one command, login and discover access</span></li>
            <li className='flex items-start'><img src='/images/check.svg' className='w-5 h-5 mr-3 mt-1' /><span className='text-left'>Short-lived, auto-refreshed credentials under the hood</span></li>
            <li className='flex items-start'><img src='/images/check.svg' className='w-5 h-5 mr-3 mt-1' /><span className='text-left'>Nothing else to set up for your team</span></li>
            <li className='flex items-start pt-3'><img src='/images/kubernetes.svg' className='w-[21px] h-[21px] mr-3' /> <span className='text-white font-bold'>Available for Kubernetes</span></li>
          </ul>
          <div className='relative z-40 flex flex-col my-16 lg:my-10 mx-auto max-w-sm lg:mx-none lg:max-w-xl lg:mx-0 xl:flex-row flex-1 items-stretch xl:items-start justify-center xl:justify-start space-y-6 xl:space-y-0 xl:space-x-4'>
            <Link href='https://github.com/infrahq/infra'>
              <div className='flex-none inline-flex rounded-full overflow-hidden bg-gradient-to-tr from-cyan-100 to-pink-300 group'>
                <button className='w-full justify-center flex m-0.5 pr-5 pl-3 py-2 text-lg rounded-full bg-black text-gray-100 transition-all duration-300 group-hover:bg-gray-900 group-hover:text-white'><img className='pr-3' src='/images/github.svg' /> Open in GitHub</button>
              </div>
            </Link>
            <div className='flex-1 flex md:mr-0'>
              <SignupForm />
            </div>
          </div>
        </div>
        <div className='flex flex-1 relative items-center justify-center max-w-5xl'>
          <img className='block relative top-24 md:top-16 lg:-right-12 scale-125 md:scale-100 lg:scale-125' src='https://user-images.githubusercontent.com/251292/174117176-680ee285-dd39-4a0b-a0b8-b75a09ccdc2e.png' />
        </div>
      </section>
    </>
  )
}

Index.layout = page => (
  <Layout>
    {page}
  </Layout>
)
