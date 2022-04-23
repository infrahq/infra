import useSWR, { useSWRConfig } from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'
import { useTable } from 'react-table'
import dayjs from 'dayjs'
import { ShareIcon } from '@heroicons/react/outline'

import Dashboard from '../../components/dashboard'
import InfoModal from '../../components/modals/info'
import GrantAccessContent from './grantAccessContent'
import Loader from '../../components/loader'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import HeaderIcon from '../../components/header-icon'
import DeleteModal from '../../components/modals/delete'

const columns = [
  {
    Header: 'Name',
    accessor: i => i,
    Cell: ({ value }) => (
      <div className='flex items-center'>
        <div className='font-medium'>{value.name}</div>
      </div>
    )
  }, {
    Header: 'Added',
    accessor: i => {
      return dayjs(i.created).fromNow()
    }
  }, {
    id: 'remove',
    accessor: i => i,
    Cell: ({ row }) => {
      const [open, setOpen] = useState(false)
      const { mutate } = useSWRConfig()

      const { id, name } = row.original

      return (
        <div className='flex justify-end text-right'>
          <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer'>
            <div className='flex items-center p-2 text-gray-500 hover:text-white'>
              <ShareIcon className='w-6 h-6 ' /><div className='text-sm ml-1'>Remove</div>
            </div>
            <DeleteModal
              open={open}
              setOpen={setOpen}
              onSubmit={() => {
                fetch(`/v1/destinations/${id}`, { method: 'DELETE' })
                  .then(() => setOpen(false))
                  .finally(() => mutate('/v1/destinations'))
                  .catch((error) => {
                    console.log(error)
                  })
              }}
              title='Remove Cluster'
              message={name}
            />
          </button>
        </div>
      )
    }
  }, {
    id: 'grant',
    accessor: i => i,
    Cell: ({ row }) => {
      const [open, setOpen] = useState(false)

      return (
        <div className='flex justify-end text-right'>
          <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer'>
            <div className='flex items-center p-2 text-gray-500 hover:text-white'>
              <ShareIcon className='w-6 h-6 ' /><div className='text-sm ml-1'>Grant</div>
            </div>

          </button>
          <InfoModal header='Grant' handleCloseModal={() => setOpen(false)} modalOpen={open} iconPath='/grant-access-color.svg'>
            <GrantAccessContent id={row.original.id} />
          </InfoModal>
        </div>
      )
    }
  }
]

export default function () {
  const { data: destinations, error } = useSWR('/v1/destinations')
  const table = useTable({ columns, data: destinations || [] })

  const loading = !destinations && !error

  return (
    <Dashboard>
      <Head>
        <title>Destinations - Infra</title>
      </Head>
      {loading
        ? (<Loader />)
        : (
          <div className='flex flex-row mt-4 lg:mt-6'>
            {destinations?.length > 0 && (
              <div className='mt-2 mr-8'>
                <HeaderIcon iconPath='/destinations-color.svg' />
              </div>
            )}
            <div className='flex-1 flex flex-col space-y-4'>
              {destinations?.length > 0 && (
                <div className='flex justify-between items-center'>
                  <h1 className='text-2xl font-bold mt-6 mb-4'>Clusters</h1>
                  <Link href='/destinations/add'>
                    <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2'>
                      <div className='bg-black rounded-full flex items-center text-sm px-4 py-1.5 '>
                        Add Clusters
                      </div>
                    </button>
                  </Link>
                </div>
              )}
              {error?.status
                ? <div className='my-20 text-center font-light text-gray-400 text-2xl'>{error?.info?.message}</div>
                : destinations.length === 0
                  ? <EmptyTable
                      title='There are currently no clusters'
                      subtitle='There are currently no clusters'
                      iconPath='/destinations-color.svg'
                      buttonHref='/destinations/add'
                      buttonText='Add Clusters'
                    />
                  : <Table {...table} />}
            </div>
          </div>
          )}
    </Dashboard>
  )
}
