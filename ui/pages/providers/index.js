import useSWR, { useSWRConfig } from 'swr'
import { useState } from 'react'
import Head from 'next/head'
import Link from 'next/link'
import { useTable } from 'react-table'
import { XIcon } from '@heroicons/react/outline'
import dayjs from 'dayjs'

import DeleteModal from '../../components/modals/delete'
import Dashboard from '../../components/dashboard'
import Table from '../../components/table'
import Loader from '../../components/loader'

function kind (url) {
  if (url?.endsWith('.okta.com')) {
    return 'okta'
  }

  return ''
}

const columns = [{
  Header: 'Identity Provider',
  accessor: p => p,
  Cell: ({ value: provider }) => (
    <div className='flex items-center'>
      <div className='w-10 h-10 mr-4 bg-purple-100/10 font-bold rounded-lg flex items-center justify-center'>
        {kind(provider.url)
          ? (
            <img className='h-2.5' src={`/${kind(provider.url)}.svg`} />
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
          <div className='flex flex-row mt-10 lg:mt-20'>
            {data?.length > 0 && (
              <div className='hidden lg:flex self-start mt-4 mr-8 bg-gradient-to-br from-violet-400/30 to-pink-200/30 items-center justify-center rounded-full'>
                <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
                  <img className='w-8 h-8' src='/providers-color.svg' />
                </div>
              </div>
            )}
            <div className='flex-1 flex flex-col space-y-4'>
              {data?.length > 0 && (
                <div className='flex justify-between items-center'>
                  <h1 className='text-2xl font-bold my-4'>Identity Providers</h1>
                  <Link href='/providers/add'>
                    <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2'>
                      <div className='bg-black rounded-full flex items-center text-sm px-4 py-1.5 '>
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
                    <div className='flex flex-col text-center my-24'>
                      <div className='flex bg-gradient-to-br from-violet-400/30 to-pink-200/30 items-center justify-center rounded-full mx-auto my-4'>
                        <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
                          <img className='w-8 h-8' src='/providers-color.svg' />
                        </div>
                      </div>
                      <h1 className='text-white text-lg font-bold mb-2'>There are no identity providers</h1>
                      <h2 className='text-gray-300 mb-4 text-sm max-w-xs mx-auto'>Identity providers allow you to connect your existing users &amp; groups to Infra.</h2>
                      <Link href='/providers/add'>
                        <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2 mx-auto'>
                          <div className='bg-black rounded-full flex items-center tracking-tight text-sm px-6 py-3 '>
                            Add Identity Provider
                          </div>
                        </button>
                      </Link>
                    </div>
                    )
                  : <Table {...table} />}
            </div>
          </div>
          )}
    </Dashboard>
  )
}
