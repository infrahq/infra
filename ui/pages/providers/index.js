import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'

import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'

export default function Providers() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 999
  const { data: { items: providers } = {} } = useSWR(
    `/api/providers?page=${page}&limit=${limit}`
  )

  return (
    <div className='mb-10'>
      <Head>
        <title>Providers - Infra</title>
      </Head>

      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 font-display text-xl font-medium'>Providers</h1>
        <Link href='/providers/add'>
          <a className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'>
            Connect provider
          </a>
        </Link>
      </header>

      <Table
        href={row => `/providers/${row.original.id}`}
        data={providers}
        empty='No providers'
        columns={[
          {
            cell: info => (
              <div className='flex flex-row items-center py-1'>
                <div className='mr-3 flex h-9 w-9 flex-none items-center justify-center rounded-md border border-gray-200'>
                  <img
                    alt='provider icon'
                    className='h-4'
                    src={`/providers/${info.row.original.kind}.svg`}
                  />
                </div>
                <div className='flex flex-col'>
                  <div className='text-sm font-medium text-gray-700'>
                    {info.getValue()}
                  </div>
                  <div className='text-2xs text-gray-500 sm:hidden'>
                    {info.row.original.url}
                  </div>
                  <div className='font-mono text-2xs text-gray-400 lg:hidden'>
                    {info.row.original.clientID}
                  </div>
                </div>
              </div>
            ),
            header: () => <span>Name</span>,
            accessorKey: 'name',
          },
          {
            cell: info => (
              <div className='hidden sm:table-cell'>{info.getValue()}</div>
            ),
            header: () => <span className='hidden sm:table-cell'>URL</span>,
            accessorKey: 'url',
          },
          {
            cell: info => (
              <div className='hidden font-mono lg:table-cell'>
                {info.getValue()}
              </div>
            ),
            header: () => (
              <span className='hidden lg:table-cell'>Client ID</span>
            ),
            accessorKey: 'clientID',
          },
        ]}
      />
    </div>
  )
}

Providers.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
