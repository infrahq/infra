import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import Head from 'next/head'
import { useTable } from 'react-table'
import { XIcon } from '@heroicons/react/outline'

import { kind } from '../../lib/providers'

import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/modals/delete'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/page-header'

const columns = [{
  Header: 'Name',
  accessor: p => p,
  Cell: ({ value: provider }) => (
    <div className='flex py-1.5 items-center'>
      <div className='border border-gray-800 flex-none flex items-center justify-center w-7 h-7 rounded-md'>
        {kind(provider.url)
          ? <img className='h-2' src={`/providers/${kind(provider.url)}.svg`} />
          : provider.name[0].toUpperCase()}
      </div>
      <div className='text-2xs leading-none ml-3'>{provider.name}</div>
    </div>
  )
}, {
  Header: 'URL',
  accessor: p => p,
  Cell: ({ value: provider }) => (
    <div className='text-3xs text-gray-400 font-mono'>{provider.url}</div>
  )
}, {
  id: 'delete',
  accessor: p => p,
  Cell: ({ value: provider, rows }) => {
    const { mutate } = useSWRConfig()

    const [open, setOpen] = useState(false)

    return (
      <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
        <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer'>
          <XIcon className='w-4 h-4 text-gray-500 hover:text-white' />
        </button>
        <DeleteModal
          open={open}
          setOpen={setOpen}
          onCancel={() => setOpen(false)}
          onSubmit={() => {
            mutate('/api/providers', async providers => {
              await fetch(`/api/providers/${provider.id}`, {
                method: 'DELETE'
              })

              return providers?.filter(p => p?.id !== provider.id)
            }, { optimisticData: rows.map(r => r.original).filter(p => p?.id !== provider.id) })

            setOpen(false)
          }}
          title='Remove Identity Provider'
          message={(<>Are you sure you want to delete <span className='font-bold text-white'>{provider.name}</span>? This action cannot be undone.</>)}
        />
      </div>
    )
  }
}]

export default function Providers () {
  const { data: { items: providers } = {}, error } = useSWR('/api/providers')

  const table = useTable({ columns, data: providers || [] })

  const loading = !providers && !error

  return (
    <>
      <Head>
        <title>Identity Providers - Infra</title>
      </Head>
      {!loading && (
        <div className='flex-1 flex flex-col space-y-4'>
          <PageHeader header='Providers' buttonHref='/providers/add' buttonLabel='Provider' />
          {error?.status
            ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
            : (
              <>
                <Table highlight={false} {...table} />
                {providers?.length === 0 &&
                  <EmptyTable
                    title='There are no providers'
                    subtitle={<>Identity providers allow you to connect your existing users &amp; groups to Infra.</>}
                    iconPath='/providers.svg'
                    buttonHref='/providers/add'
                    buttonText='Provider'
                  />}
              </>
              )}
        </div>
      )}
    </>
  )
}

Providers.layout = function (page) {
  return (
    <Dashboard>
      {page}
    </Dashboard>
  )
}
