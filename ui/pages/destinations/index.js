import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'
import dayjs from 'dayjs'

import Dashboard from '../../components/dashboard/dashboard'
import InfoModal from '../../components/modals/infoModal'
import GrantAccessContent from './grantAccessContent'
import Loader from '../../components/loader'
import Table from '../../components/table'
import EmptyTable from '../../components/emptyTable'
import HeaderIcon from '../../components/dashboard/headerIcon'

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
                  <h1 className='text-2xl font-bold mt-6 mb-4'>Destinations</h1>
                  <Link href='/destinations/add/details'>
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
                  ? <EmptyTable
                      title='There are no destinations'
                      subtitle='TODO TODO'
                      iconPath='/destinations-color.svg'
                      buttonHref='/destinations/add/details'
                      buttonText='Add Destination'
                    />
                  : <Table
                      columns={columns}
                      data={destinations || []}
                      getRowProps={row => ({
                        onClick: () => handleDestinationDetail(row.original.id),
                        style: {
                          cursor: 'pointer'
                        }
                      })}
                    />}
            </div>
            <InfoModal header='Grant' handleCloseModal={() => setModalOpen(false)} modalOpen={modalOpen} iconPath='/grant-access-color.svg'>
              <GrantAccessContent id={SelectedId} />
            </InfoModal>
          </div>
          )}
    </Dashboard>
  )
}
