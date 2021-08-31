import Head from 'next/head'
import useSWR from 'swr'

import { InfoApi, Configuration } from '../api'
import Dashboard from '../layouts/Dashboard'

export default function Settings () {
  const { data: version } = useSWR(
    'version',
    () => new InfoApi(new Configuration({ basePath: '/v1' })).version()
  )

  return (
    <Dashboard>
      <Head>
        <title>Settings – Infra</title>
        <meta property="og:title" content="Infrastructure - Infra" key="title" />
      </Head>
      <div className="flex flex-col bg-white rounded-lg shadow mt-8">
        <div className="flex justify-between items-center pl-6 pr-4 border-b">
          <h1 className="text-md font-semibold text-black py-4">General</h1>
        </div>
        <div className="px-4 py-5 sm:p-0">
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Version</dt>
            <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">{version?.version}</dd>
          </div>
        </div>
      </div>
    </Dashboard>
  )
}
