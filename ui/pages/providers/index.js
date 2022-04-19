import useSWR from 'swr'
import { useState } from 'react'
import Head from 'next/head'
import Link from 'next/link'
import { useTable } from 'react-table'
import { XIcon, PlusIcon } from '@heroicons/react/outline'

import Dashboard from '../../components/dashboard'
import Table from '../../components/table'

const columns = [
  {
    Header: 'Name',
    accessor: 'name',
    Cell: ({ value }) => (
      <div className='flex items-center'>
        <div className='w-12 h-12 mr-4 bg-purple-100/10 font-bold rounded-xl flex items-center justify-center'><img className='h-3' src='/okta.svg' /></div>
        <div>{value}</div>
      </div>
    )
  },
  {
    accessor: 'url', // accessor is the "key" in the data,
    Header: () => (
      <div className='text-right'>
        URL
      </div>
    ),
    Cell: ({ value }) => (
      <div className='text-right'>
        {value}
      </div>
    )
  }, {
    id: 'delete',
    accessor: (r) => r,
    Cell: ({ value: provider }) => {
      const [open, setOpen] = useState(false)
      return (
        <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
          <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer'>
            <XIcon className='w-5 h-5 text-gray-500' />
          </button>
          <DeleteModal
            open={open}
            setOpen={setOpen}
            title='Delete Identity Provider'
            message={(<>Are you sure you want to delete <span className='font-bold'>{provider.name}</span>? This action cannot be undone.</>)}
          />
        </div>
      )
    }
  }
]

export default function () {
  const { data, error } = useSWR('/v1/providers', { fallbackData: [] })

  const table = useTable({ columns, data })

  return (
    <Dashboard>
      <Head>
        <title>Identity Providers - Infra</title>
      </Head>
      <div className='flex flex-col my-20'>
        <h1 className='text-4xl font-bold my-8'>Identity Providers</h1>
        {error?.status
          ? <div className='my-20 text-center font-light text-gray-400 text-2xl'>{error?.info?.message}</div>
          : data.length === 0
            ? (
              <div className='text-center my-20'>
                <p className='text-gray-400 mb-4 text-2xl'>No Identity Providers</p>
                <Link href='/providers/add'>
                  <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2'>
                    <div className='bg-black rounded-full flex items-center tracking-tight px-4 py-2 '>
                      <PlusIcon className='w-5 h-5 mr-2' />Add Identity Provider
                    </div>
                  </button>
                </Link>
              </div>
              )
            : <Table {...table} />}
      </div>
    </Dashboard>
  )
}
