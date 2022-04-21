import Head from 'next/head'

import Dashboard from '../../components/dashboard/dashboard'
import HeaderIcon from '../../components/dashboard/headerIcon'
import Admin from './admin'

export default function () {
  return (
    <Dashboard>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex flex-row mt-4 lg:mt-6'>
        {/* TODO: icon path is wrong */}
        <div className='mt-2 mr-6'>
          <HeaderIcon width={6} iconPath='/destinations-color.svg' />
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