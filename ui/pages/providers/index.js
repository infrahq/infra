import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import Head from 'next/head'
import { useTable } from 'react-table'
import { XIcon } from '@heroicons/react/outline'
import dayjs from 'dayjs'

import { kind } from '../../lib/providers'

import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/modals/delete'
import Table from '../../components/table'
import Loader from '../../components/loader'
import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/layouts/page-header'

const columns = [{
  Header: 'Identity Provider',
  width: '55%',
  accessor: p => p,
  Cell: ({ value: provider }) => (
    <div className='flex items-center'>
      <div className='w-10 h-10 mr-4 bg-purple-100/10 font-bold rounded-lg flex items-center justify-center'>
        {kind(provider.url)
          ? (
            <img className='h-2.5' src={`/providers/${kind(provider.url)}.svg`} />
            )
          : (
              provider.name[0].toUpperCase()
            )}
      </div>
      <div className='flex flex-col leading-tight'>
        <div>{provider.name}</div>
        <div className='text-gray-300 text-xs'>{provider.url}</div>
      </div>
    </div>
  )
}, {
  Header: 'Added',
  accessor: p => {
    return dayjs(p.created).fromNow()
  }
}, {
  id: 'delete',
  accessor: p => p,
  Cell: ({ value: provider, rows }) => {
    const { mutate } = useSWRConfig()

    const [open, setOpen] = useState(false)

    return (
      <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
        <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer'>
          <XIcon className='w-5 h-5 text-gray-300 hover:text-white' />
        </button>
        <DeleteModal
          open={open}
          setOpen={setOpen}
          onSubmit={() => {
            mutate('/v1/providers', async providers => {
              await fetch(`/v1/providers/${provider.id}`, {
                method: 'DELETE'
              })

              return providers?.filter(p => p?.id !== provider.id)
            }, { optimisticData: rows.map(r => r.original).filter(p => p?.id !== provider.id) })

            setOpen(false)
          }}
          title='Delete Identity Provider'
          message={(<>Are you sure you want to delete <span className='font-bold text-white'>{provider.name}</span>? This action cannot be undone.</>)}
        />
      </div>
    )
  }
}]

export default function Providers () {
  const { data, error } = useSWR('/v1/providers')

  const table = useTable({ columns, data: data || [] })

  const loading = !data && !error

  return (
    <>
      <Head>
        <title>Identity Providers - Infra</title>
      </Head>
      {loading
        ? (<Loader />)
        : (
          <div className='flex-1 flex flex-col space-y-8 mt-3 mb-4'>
            <PageHeader header='Providers' buttonHref='/providers/add' buttonLabel='Provider' />
            {error?.status
              ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
              :
              <>
                <Table {...table} />
                {data?.count === 0 && 
                  <EmptyTable
                    title='There are no providers'
                    subtitle={<>Identity providers allow you to connect your existing users &amp; groups to Infra.</>}
                    iconPath='/providers.svg'
                    buttonHref='/providers/add'
                    buttonText='Provider'
                  />}
              </>
              }
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
