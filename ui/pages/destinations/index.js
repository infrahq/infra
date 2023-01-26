import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'

import { useUser } from '../../lib/hooks'
import { CommandLineIcon } from '@heroicons/react/24/solid'
import { TrashIcon, PlusIcon } from '@heroicons/react/24/outline'

import { RemoveButtonType } from '../../lib/type'

import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'
import RemoveButton from '../../components/remove-button'

export default function Destinations() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 20

  const { isAdmin, isAdminLoading } = useUser()

  const { data: { items: destinations, totalCount, totalPages } = {}, mutate } =
    useSWR(`/api/destinations?page=${page}&limit=${limit}`)

  if (isAdminLoading) {
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
            <PlusIcon className='mr-1 h-3 w-3' /> Connect Cluster
          </Link>
        )}
      </header>

      <Table
        href={row =>
          row.original.kind === 'kubernetes' &&
          `/destinations/${row.original.id}`
        }
        count={totalCount}
        pageCount={totalPages}
        pageIndex={parseInt(page) - 1}
        pageSize={limit}
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
            cell: info => (
              <div className='flex flex-row items-center py-1'>
                <div className='mr-3 flex h-9 w-9 flex-none items-center justify-center rounded-md border border-gray-200'>
                  {info.row.original.kind === 'ssh' ? (
                    <CommandLineIcon className='h-5 text-black' />
                  ) : (
                    <img
                      alt='kubernetes icon'
                      className='h-5'
                      src={`/kubernetes.svg`}
                    />
                  )}
                </div>
                <div className='flex flex-col'>
                  <div className='text-sm font-medium text-gray-700'>
                    {info.getValue()}
                  </div>

                  <div className='text-2xs text-gray-500'>
                    {info.row.original.connection.url === ''
                      ? '-'
                      : info.row.original.connection.url}
                  </div>
                </div>
              </div>
            ),
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
            id: 'delete',
            cell: function Cell(info) {
              return (
                info.row.original.kind === 'ssh' && (
                  <RemoveButton
                    onRemove={async () => {
                      await fetch(`/api/destinations/${info.row.original.id}`, {
                        method: 'DELETE',
                      })

                      mutate()
                    }}
                    type={RemoveButtonType.Link}
                    modalTitle='Remove destination'
                    modalMessage={
                      <div>
                        Are you sure you want to remove{' '}
                        <span className='break-all font-bold'>
                          {info.row.original.name}
                        </span>
                        ?
                      </div>
                    }
                  >
                    <div className='flex flex-row items-center'>
                      <TrashIcon className='mr-2 h-3.5 w-3.5' />
                      Remove
                      <span className='sr-only'>{info.row.original.name}</span>
                    </div>
                  </RemoveButton>
                )
              )
            },
          },
        ]}
      />
    </div>
  )
}

Destinations.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
