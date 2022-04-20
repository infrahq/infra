import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'
import dayjs from 'dayjs'

import Dashboard from '../../components/dashboard'
import Modal from '../../components/modal'
import GrantAccessContent from '../../components/grantAccessContent'
import Loader from '../../components/loader'
import Table from '../../components/table'

const columns = [
  {
    Header: 'Destination',
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
  }
]

export default function () {
  const { data: destinations, error } = useSWR('/v1/destinations')

  const [modalOpen, setModalOpen] = useState(false)
  const [SelectedId, setSelectedId] = useState(null)

  const loading = !destinations && !error

  const handleDestinationDetail = (id) => {
    setSelectedId(id)
    setModalOpen(true)
  }

  return (
    <Dashboard>
      <Head>
        <title>Destinations - Infra</title>
      </Head>
      {loading
        ? ( <Loader /> )
        : (
          <div className='flex flex-row mt-4 lg:mt-6'>
          {destinations?.length > 0 && (
            <div className='hidden lg:flex self-start mt-2 mr-8 bg-gradient-to-br from-violet-400/30 to-pink-200/30 items-center justify-center rounded-full'>
              <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
                <img className='w-8 h-8' src='/destinations-color.svg' />
              </div>
            </div>
          )}
          <div className='flex-1 flex flex-col space-y-4'>
            {destinations?.length > 0 && (
              <div className='flex justify-between items-center'>
                <h1 className='text-2xl font-bold mt-6 mb-4'>Destinations</h1>
                <Link href='/destinations/add/connect'>
                  <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2'>
                    <div className='bg-black rounded-full flex items-center text-sm px-4 py-1.5 '>
                      Add Destination
                    </div>
                  </button>
                </Link>
              </div>
            )}
            {error?.status
              ? <div className='my-20 text-center font-light text-gray-400 text-2xl'>{error?.info?.message}</div>
              : destinations.length === 0
                ? (
                  <div className='flex flex-col text-center my-24'>
                    <div className='flex bg-gradient-to-br from-violet-400/30 to-pink-200/30 items-center justify-center rounded-full mx-auto my-4'>
                      <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
                        <img className='w-8 h-8' src='/destinations-color.svg' />
                      </div>
                    </div>
                    <h1 className='text-white text-lg font-bold mb-2'>There are no destination connected</h1>
                    <h2 className='text-gray-300 mb-4 text-sm max-w-xs mx-auto'>TODO: WE NEED TEXT HERE</h2>
                    <Link href='/destinations/add/connect'>
                      <button className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 my-2 mx-auto'>
                        <div className='bg-black rounded-full flex items-center tracking-tight text-sm px-6 py-3'>
                          Add Destination
                        </div>
                      </button>
                    </Link>
                  </div>
                  )
                : <Table
                    columns={columns}
                    data={destinations || []}
                    getRowProps={row => ({
                      onClick: () => handleDestinationDetail(row.original.id),
                      style: {
                        cursor: "pointer"
                      }
                    })}
                  />}
            </div>
            <Modal header='Grant' handleCloseModal={() => setModalOpen(false)} modalOpen={modalOpen} iconPath='/grant-access-color.svg'>
              <GrantAccessContent id={SelectedId} />
            </Modal>
          </div>
      )}
    </Dashboard>
  )
}
