import useSWR, { useSWRConfig } from 'swr'
import Head from 'next/head'
import { useState } from 'react'
import { useTable } from 'react-table'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import Loader from '../../components/loader'
import Table from '../../components/table'
import EmptyTable from '../../components/empty-table'
import DeleteModal from '../../components/modals/delete'
import Grant from '../../components/grant'
import PageHeader from '../../components/layouts/page-header'
import Slide from '../../components/slide'

const columns = [{
  Header: 'Name',
  accessor: 'name',
  Cell: ({ value }) => (
    <div className='flex items-center'>
      <div className='py-2 flex items-center'><img className='opacity-25 mr-4' src='/infrastructure.svg' /> {value}</div>
    </div>
  )
}, {
  Header: 'Kind',
  id: 'kind',
  Cell: 'Cluster'
}, {
  id: 'connected',
  Header: () => (
    <div className='text-right'>Connection</div>
  ),
  accessor: 'updated',
  Cell: ({ value: updated }) => (
    <div className='flex items-center justify-end'>
      <div className='w-[7px] h-[7px] bg-green-400 rounded-full mr-2' />
      {new Date() - new Date(updated)}
    </div>
  )
}]

function SlideContent ({ id, isAdmin }) {
  const { data: destination } = useSWR(`/v1/destinations/${id}`)
  return (
    <>
      {isAdmin &&
        <>
          <div className='border-b border-gray-800 mt-4'>
            <div className='text-label text-gray-400 uppercase pb-5'>Access</div>
          </div>
          <div className='pt-3 pb-12'>
            <Grant id={id} />
          </div>
        </>}
      <>
        <div className='border-b border-gray-800 mt-4'>
          <div className='text-label text-gray-400 uppercase pb-5'>Meta</div>
        </div>
        <div className='pt-3 flex flex-col space-y-2'>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-name w-1/3'>Kind</div>
            <div className='text-name' />
          </div>
          <div className='flex flex-row flex-start'>
            <div className='text-gray-400 text-name w-1/3'>Namespace</div>
            <div className='flex flex-col'>
              {destination?.resources.map(r => (
                <div key={r} className='text-name'>{r}</div>
              ))}
            </div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-name w-1/3'>Age</div>
            <div className='text-name'>{dayjs(destination?.created).fromNow()}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-name w-1/3'>Images</div>
            <div className='text-name' />
          </div>
        </div>
      </>
    </>
  )
}

export default function Destinations () {
  const { data: destinations, error } = useSWR('/v1/destinations')
  const { mutate } = useSWRConfig()
  const { admin, loading: adminLoading } = useAdmin()
  const [DeleteModalOpen, setDeleteModalOpen] = useState(false)
  const [slideModalOpen, setSlideModalOpen] = useState(false)
  const [selectedRow, setSelectedRow] = useState(null)
  const [slideActionBtns, setSlideActionBtns] = useState([])

  const table = useTable({ columns, data: destinations || [] })

  const loading = adminLoading || (!destinations && !error)

  const handleDestinationDetail = (row) => {
    setSlideModalOpen(true)
    setSelectedRow(row)
    setSlideActionBtns([{ handleOnClick: () => setDeleteModalOpen(true), text: 'Disconnect Cluster' }])
  }

  const handleCancelDeleteModal = () => {
    setDeleteModalOpen(false)
    setSlideModalOpen(true)
  }

  return (
    <>
      <Head>
        <title>Destinations - Infra</title>
      </Head>
      {loading
        ? (<Loader />)
        : (
          <div className={`flex-1 flex flex-col space-y-8 mt-3 mb-4 ${slideModalOpen ? 'w-7/12' : ''}`}>
            <PageHeader header='Infrastructure' buttonHref={admin && '/destinations/add'} buttonLabel='Infrastructure' />
            {error?.status
              ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
              : <>
                <Table
                  {...table}
                  getRowProps={row => ({
                    onClick: () => handleDestinationDetail(row),
                    style: {
                      cursor: 'pointer'
                    }
                  })}
                />
                <>
                  {slideModalOpen &&
                    <Slide open={slideModalOpen} handleClose={() => setSlideModalOpen(false)} title={selectedRow.values.name} iconPath='/destinations.svg' footerBtns={slideActionBtns} deleteModalShown={DeleteModalOpen}>
                      <SlideContent id={selectedRow.original.id} isAdmin={admin} />
                    </Slide>}
                  <DeleteModal
                    open={DeleteModalOpen}
                    setOpen={setDeleteModalOpen}
                    onCancel={handleCancelDeleteModal}
                    onSubmit={async () => {
                      mutate('/v1/destinations', async destinations => {
                        await fetch(`/v1/destinations/${selectedRow.original.id}`, {
                          method: 'DELETE'
                        })

                        return destinations?.filter(d => d?.id !== selectedRow.original.id)
                      })

                      setDeleteModalOpen(false)
                    }}
                    title='Delete Cluster'
                    message={<>Are you sure you want to disconnect <span className='text-white font-bold'>{selectedRow?.original.name}?</span><br />Note: you must also uninstall the Infra Connector from this cluster.</>}
                  />
                </>
                {
                    destinations?.length === 0 &&
                      <EmptyTable
                        title='There are no infrastructure'
                        subtitle={`There are currently no infrastructure connected to Infra. ${admin ? 'Get started by connecting one.' : ''}`}
                        iconPath='/destinations.svg'
                        buttonHref={admin && '/destinations/add'}
                        buttonText='Infrastructure'
                      />
                  }
              </>}
          </div>
          )}
    </>
  )
}

Destinations.layout = function (page) {
  return (
    <Dashboard>
      {page}
    </Dashboard>
  )
}
