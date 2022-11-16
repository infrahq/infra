import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'

import { useUser } from '../../lib/hooks'

import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'

const LIMIT = 20

export default function Destinations() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p

  const { user, isAdmin, loading } = useUser()

  const { data: { items: destinations, totalCount, totalPages } = {} } = useSWR(
    `/api/destinations?page=${page}&limit=${LIMIT}`
  )

  const { data: { items: currentUserGrants } = {} } = useSWR(
    `/api/grants?user=${user?.id}&limit=1000&showSystem=1`
  )

  if (loading) {
    return null
  }

  return (
    <div className='mb-10'>
      <Head>
        <title>Infrastructure - Infra</title>
      </Head>
      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 font-display text-xl font-medium'>
          Infrastructure
        </h1>

        {/* Add dialog */}
        {isAdmin && (
          <Link
            href='/destinations/add'
            className='inline-flex items-center rounded-md border border-transparent bg-black  px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
          >
            Connect cluster
          </Link>
        )}
      </header>

      <Table
        href={row => `/destinations/${row.original.id}`}
        count={totalCount}
        pageCount={totalPages}
        pageIndex={parseInt(page) - 1}
        pageSize={LIMIT}
        data={destinations}
        empty='No infrastructure'
        onPageChange={({ pageIndex }) => {
          router.push({
            pathname: router.pathname,
            query: { ...router.query, p: pageIndex + 1 },
          })
        }}
        columns={[
          {
            cell: function Cell(info) {
              const numAccess = isAdmin
                ? 0
                : currentUserGrants?.filter(
                    g =>
                      g.resource === info.row.original.name ||
                      g.resource.slice(0, info.row.original.name.length) ===
                        info.row.original.name
                  ).length

              return (
                <>
                  <div className='flex flex-row items-center py-1'>
                    <div className='mr-3 flex h-9 w-9 flex-none items-center justify-center rounded-md border border-gray-200'>
                      <img
                        alt='kubernetes icon'
                        className='h-5'
                        src={`/kubernetes.svg`}
                      />
                    </div>
                    <div className='flex flex-col'>
                      <div className='text-sm font-medium text-gray-700'>
                        {info.getValue()}
                      </div>
                      <div className='text-2xs text-gray-500'>
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
                  <div className='sm:hidden'>
                    {!isAdmin && numAccess === 0 && (
                      <span className='inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-px text-2xs font-medium text-yellow-800'>
                        No Access
                      </span>
                    )}
                  </div>
                </>
              )
            },
            header: () => <span>Name</span>,
            accessorKey: 'name',
          },
          {
            cell: info => (
              <span className='hidden lg:table-cell'>{info.getValue()}</span>
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
                      ? info.row.original.connection.url === ''
                        ? 'animate-pulse border-yellow-500 bg-yellow-500'
                        : 'border-teal-400 bg-teal-400'
                      : 'border-gray-200 bg-gray-200'
                  }`}
                />
                <span className='flex-none px-2'>
                  {info.getValue()
                    ? info.row.original.connection.url === ''
                      ? 'Pending'
                      : 'Connected'
                    : 'Disconnected'}
                </span>
              </div>
            ),
            header: () => <span>Status</span>,
            accessorKey: 'connected',
          },
          {
            cell: function Cell(info) {
              const numAccess = currentUserGrants?.filter(
                g =>
                  g.resource === info.row.original.name ||
                  g.resource.slice(0, info.row.original.name.length) ===
                    info.row.original.name
              ).length
              return (
                !isAdmin &&
                numAccess === 0 && (
                  <span className='hidden items-center rounded-full bg-yellow-100 px-2.5 py-px text-2xs font-medium text-yellow-800 sm:inline-flex'>
                    No Access
                  </span>
                )
              )
            },
            id: 'access_status',
          },
        ]}
      />
    </div>
  )
}

Destinations.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
