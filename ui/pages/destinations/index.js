import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'
import { useState } from 'react'

import { useUser } from '../../lib/hooks'
import { CommandLineIcon, TrashIcon } from '@heroicons/react/24/solid'

import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'

export default function Destinations() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 20

  const { isAdmin, loading } = useUser()

  const { data: { items: destinations, totalCount, totalPages } = {}, mutate } =
    useSWR(`/api/destinations?page=${page}&limit=${limit}`)
  const [openSelectedDeleteModal, setOpenSelectedDeleteModal] = useState(false)
  const [selectedDeleteId, setSelectedDeleteId] = useState(null)

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
                  {info.row.original.kind !== 'ssh' ? (
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
                  ) : (
                    <div className='text-2xs text-gray-500'>
                      {info.row.original.connection.url === ''
                        ? '-'
                        : info.row.original.connection.url}
                    </div>
                  )}
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
                  <div className='group invisible rounded-md bg-white group-hover:visible'>
                    <button
                      type='button'
                      onClick={() => {
                        setSelectedDeleteId(info.row.original.id)
                        setOpenSelectedDeleteModal(true)
                      }}
                      className='flex items-center text-xs font-medium text-red-500 hover:text-red-500/50'
                    >
                      <TrashIcon className='mr-2 h-3.5 w-3.5' />
                      <span className='hidden sm:block'>Remove</span>
                    </button>
                  </div>
                )
              )
            },
          },
        ]}
      />
      <DeleteModal
        open={openSelectedDeleteModal}
        setOpen={setOpenSelectedDeleteModal}
        onSubmit={async () => {
          await fetch(`/api/destinations/${selectedDeleteId}`, {
            method: 'DELETE',
          })

          mutate()
          setSelectedDeleteId(null)
          setOpenSelectedDeleteModal(false)
        }}
        title={'Remove destination'}
        message={<>Are you sure you want to remove the selected destination?</>}
      />
    </div>
  )
}

Destinations.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
