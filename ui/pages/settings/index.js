import Head from 'next/head'

import Dashboard from '../../components/layouts/dashboard'
import Admin from './admin'

export default function Settings () {
  return (
    <>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex flex-row mt-6 mb-4'>
        <div className='w-[18px] h-[19px] mr-3'>
          <img src='/settings.svg' />
        </div>
        <div className='flex-1 flex flex-col space-y-4'>
          <h1 className='text-title mb-6'>Settings</h1>
          <Admin />
        </div>
      </div>
    </>
  )
}

Settings.layout = function (page) {
  return (
    <Dashboard>
      {page}
    </Dashboard>
  )
}
