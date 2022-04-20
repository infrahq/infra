import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import Head from 'next/head'
import Link from 'next/link'
import { useTable } from 'react-table'
import { XIcon } from '@heroicons/react/outline'
import dayjs from 'dayjs'

import { kind } from '../../lib/providers'

import DeleteModal from '../../components/modals/delete'
import Dashboard from '../../components/dashboard/dashboard'
import Table from '../../components/table'
import { kind } from '../../lib/providers'

const columns = [{
  Header: 'Identity Provider',
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
        <div className='font-medium'>{provider.name}</div>
        <div className='text-gray-400 text-xs'>{provider.url}</div>
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
    console.log(rows)
    return (
      <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
        <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer'>
          <XIcon className='w-5 h-5 text-gray-500' />
        </button>
        <DeleteModal
          open={open}
          setOpen={setOpen}
          onSubmit={() => {
            mutate('/v1/providers', async providers => {
              await fetch(`/v1/providers/${provider.id}`, {
                method: 'DELETE'
              })

              return providers.filter(p => p?.id !== provider.id)
            }, { optimisticData: rows.map(r => r.original).filter(p => p?.id !== provider.id) })
          }}
          title='Delete Identity Provider'
          message={(<>Are you sure you want to delete <span className='font-bold text-white'>{provider.name}</span>? This action cannot be undone.</>)}
        />
      </div>
    )
  }
}]

export default function () {
  const { data, error } = useSWR('/v1/providers')
  const table = useTable({ columns, data: data || [] })

  const loading = !data && !error

  return (
    <Dashboard>
      <Head>
        <title>Identity Providers - Infra</title>
      </Head>
      {loading
        ? ( <Loader /> )
        : (
          <div className='flex flex-row mt-4 lg:mt-6'>
            {data?.length > 0 && (
              <HeaderIcon iconPath='/providers-color.svg' />
            )}
            <div className='flex-1 flex flex-col space-y-4'>
              {data?.length > 0 && (
                <div className='flex justify-between items-center'>
                  <h1 className='text-2xl font-bold mt-6 mb-4'>Identity Providers</h1>
                  <Link href='/providers/add'>
                    <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2'>
                      <div className='bg-black rounded-full flex items-center text-sm px-4 py-1.5'>
                        Add Identity Provider
                      </div>
                    </button>
                  </Link>
                </div>
              )}
              {error?.status
                ? <div className='my-20 text-center font-light text-gray-400 text-2xl'>{error?.info?.message}</div>
                : data.length === 0
                  ? (
                    <EmptyTable
                      title='There are no identity providers'
                      subtitle={<>Identity providers allow you to connect your existing users &amp; groups to Infra.</>}
                      iconPath='/providers-color.svg'
                      buttonHref='/providers/add'
                      buttonText='Add Identity Provider'
                    />
                    )
                  : <Table {...table} />}
            </div>
          </div>
          )}
    </Dashboard>
  )
}
