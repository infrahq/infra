import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'
import dayjs from 'dayjs'

import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'

export default function Destinations() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 20

  const { data: { items: destinations, totalCount, totalPages } = {} } = useSWR(
    `/api/destinations?page=${page}&limit=${limit}`
  )

  return (
    <div className='mb-10'>
      <Head>
        <title>Clusters - Infra</title>
      </Head>
      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 text-xl font-medium'>Clusters</h1>
        {/* Add dialog */}
        <Link href='/destinations/add'>
          <a className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'>
            Connect cluster
          </a>
        </Link>
      </header>

      <Table
        href={row => `/destinations/${row.original.id}`}
        count={totalCount}
        pageCount={totalPages}
        pageIndex={parseInt(page) - 1}
        pageSize={limit}
        data={destinations}
        empty='No clusters'
        onPageChange={({ pageIndex }) => {
          router.push({
            pathname: router.pathname,
            query: { ...router.query, p: pageIndex + 1 },
          })
        }}
        columns={[
          {
            cell: info => (
              <div className='flex flex-row py-1'>
                <div className='mr-2.5 flex h-8 w-8 flex-none items-center justify-center rounded-md border border-gray-200'>
                  <img
                    alt='kubernetes icon'
                    className='h-[18px]'
                    src={`/kubernetes.svg`}
                  />
                </div>
                <div className='flex flex-col'>
                  <div className='mb-0.5 font-medium text-gray-700'>
                    {info.getValue()}
                  </div>
                  <div className='text-3xs text-gray-500'>
                    {info.row.original.resources?.length > 0 && (
                      <span>
                        {info.row.original.resources?.length}&nbsp;
                        {info.row.original.resources?.length === 1
                          ? 'namespace'
                          : 'namespaces'}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            ),
            header: () => <span>Name</span>,
            accessorKey: 'name',
          },
          {
            cell: info => (
              <div className='hidden truncate lg:table-cell'>
                {info.getValue() ? dayjs(info.getValue()).fromNow() : '-'}
              </div>
            ),
            header: () => <span className='hidden lg:table-cell'>Added</span>,
            accessorKey: 'created',
          },
          {
            cell: info => (
              <span className='hidden font-mono text-2xs lg:table-cell'>
                {info.getValue()}
              </span>
            ),
            header: () => (
              <span className='hidden lg:table-cell'>Connector Version</span>
            ),
            accessorKey: 'version',
          },
          {
            cell: info => (
              <div className='flex items-center py-2'>
                <div
                  className={`h-2 w-2 flex-none rounded-full border ${
                    info.getValue()
                      ? ' border-teal-500/50 bg-teal-400'
                      : 'border-gray-200 bg-gray-100'
                  }`}
                />
                <span className='flex-none px-2'>
                  {info.getValue() ? 'Connected' : 'Disconnected'}
                </span>
              </div>
            ),
            header: () => <span>Status</span>,
            accessorKey: 'connected',
          },
        ]}
      />
    </div>
  )
}

Destinations.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
