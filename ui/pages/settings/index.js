import Head from 'next/head'

import Dashboard from '../../components/dashboard'
import Admin from './admin'

export default function () {
  return (
    <Dashboard>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex flex-row mt-4 lg:mt-6'>
        <div className='hidden lg:flex self-start mt-2 mr-8 bg-gradient-to-br from-violet-400/30 to-pink-200/30 items-center justify-center rounded-full'>
          <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
            <img className='w-8 h-8' src='/destinations-color.svg' />
          </div>
        </div>
        <div className='flex-1 flex flex-col space-y-4'>
          <h1 className='text-2xl font-bold mt-6 mb-4'>Settings</h1>
          <div className='pt-3'>
            <Admin />
          </div>
        </div>
      </div>
    </Dashboard>   
  )
}