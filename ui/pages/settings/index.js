import Head from 'next/head'

import Dashboard from '../../components/layouts/dashboard'
import Admin from './admin'

export default function Settings () {
  return (
    <>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex-1 flex flex-col space-y-8 mt-6 mb-4'>
        <h1 className='text-title mb-6 font-bold'>Settings</h1>
        <Admin />
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
