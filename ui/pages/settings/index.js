import Head from 'next/head'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import Admin from '../../components/settings/admin'
import Account from '../../components/settings/account'

export default function Settings () {
  const { admin, loading } = useAdmin()

  return (
    <>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      {!loading &&(
        <div className='flex-1 flex flex-col space-y-8 mt-6 mb-4'>
          <h1 className='text-xs mb-6 font-bold'>Settings</h1>
          <Account />
          {admin && <Admin />}
        </div>
      )}
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
