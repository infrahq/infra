import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'
import { useState } from 'react'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'

export default function Providers() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 999
  const { data: { items: providers } = {}, mutate } = useSWR(
    `/api/providers?page=${page}&limit=${limit}`
  )
  const { admin } = useAdmin()

  return (
    <div className='mb-10'>
      <Head>
        <title>Providers - Infra</title>
      </Head>

      <header className='my-6 flex items-center justify-between'>
        <h1 className='py-1 text-xl font-medium'>Providers</h1>
        {admin && (
          <Link href='/providers/add' data-testid='page-header-button-link'>
            <button className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800'>
              Connect provider
            </button>
          </Link>
        )}
      </header>

      <Table
        data={providers}
        empty='No providers'
        columns={[
          {
            cell: info => (
              <div className='flex items-center font-medium text-gray-700'>
                <div className='mr-2 flex h-8 w-8 flex-none items-center justify-center rounded-md border border-gray-200'>
                  <img
                    alt='provider icon'
                    className='h-4'
                    src={`/providers/${info.row.original.kind}.svg`}
                  />
                </div>
                {info.getValue()}
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
              <div
                className='hidden truncate lg:table-cell'
                title={info.getValue()}
              >
                {info.getValue()}
              </div>
            ),
            header: () => <span className='hidden lg:table-cell'>URL</span>,
            accessorKey: 'url',
          },
          {
            cell: info => (
              <div
                className='hidden truncate font-mono lg:table-cell'
                title={info.getValue()}
              >
                {info.getValue()}
              </div>
            ),
            header: () => (
              <span className='hidden lg:table-cell'>Client ID</span>
            ),
            accessorKey: 'clientID',
          },
          {
            cell: function Cell(info) {
              const [open, setOpen] = useState(false)
              return (
                <div className='text-right'>
                  <button
                    onClick={() => setOpen(true)}
                    className='p-1 text-2xs text-gray-500/75 hover:text-gray-600'
                  >
                    Remove
                    <span className='sr-only'>{info.row.original.name}</span>
                  </button>
                  <DeleteModal
                    open={open}
                    setOpen={setOpen}
                    onSubmit={async () => {
                      await fetch(`/api/providers/${info.row.original.id}`, {
                        method: 'DELETE',
                      })
                      setOpen(false)

                      mutate({
                        items: providers.filter(
                          p => p.id !== info.row.original.id
                        ),
                      })
                    }}
                    title='Remove Identity Provider'
                    message={
                      <>
                        Are you sure you want to remove{' '}
                        <span className='font-bold'>
                          {info.row.original.name}
                        </span>
                        ?
                      </>
                    }
                  />
                </div>
              )
            },
            id: 'delete',
          },
        ]}
      />
    </div>
  )
}

Providers.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
