import Head from 'next/head'
import { useState } from 'react'
import { useTable } from 'react-table'
import useSWR, { mutate } from 'swr'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/layouts/page-header'
import Loader from '../../components/loader'
import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'
import Slide from '../../components/slide'
import ProfileIcon from '../../components/profile-icon'
import DeleteModal from '../../components/modals/delete'
import ResourcesGrant from '../../components/resources-grant'

const columns = [{
  Header: 'Name',
  accessor: u => u,
  Cell: ({ value: user }) => (
    <div className='flex items-center space-x-4'>
      <ProfileIcon name={user.name[0]} />
      <div className='flex flex-col leading-tight'>
        <div className='text-subtitle'>{user.name}</div>
      </div>
    </div>
  )
}, {
  Header: 'Last Seen',
  accessor: u => u,
  Cell: ({ value: user }) => (
    <div className='text-name text-gray-400'>{user.lastSeenAt ? dayjs(user.lastSeenAt).fromNow() : '-'}</div>
  )
}, {
  Header: 'Added',
  accessor: u => u,
  Cell: ({ value: user }) => (
    <div className='text-name text-gray-400'>{dayjs(user.created).fromNow()}</div>
  )
}]

function SlideContent ({ id, isAdmin }) {
  const { data: user } = useSWR(`/v1/identities/${id}`)
  const { data: grants } = useSWR(`/v1/identities/${id}/grants`)

  return (
    <div className='pl-4'>
      {isAdmin &&
        <>
          <div className='border-b border-gray-800 mt-4'>
            <div className='text-label text-gray-400 uppercase pb-5'>Access</div>
          </div>
          <div className='pt-3 pb-12'>
            <ResourcesGrant id={id} />
          </div>
        </>}
      <>
        <div className='border-b border-gray-800 mt-4'>
          <div className='text-label text-gray-400 uppercase pb-5'>Meta</div>
        </div>
        <div className='pt-3 flex flex-col space-y-2'>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-name w-1/3'>User</div>
            <div className='text-name'>{user?.name}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-name w-1/3'>Infra Role</div>
            <div className='text-name'>{grants?.filter(g => g.resource === 'infra').find(g => g.privilege === 'admin') ? 'Admin' : 'View'}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-name w-1/3'>Created</div>
            <div className='text-name'>{dayjs(user?.created).fromNow()}</div>
          </div>
        </div>
      </>
    </div>
  )
}

export default function Users () {
  const { data: users, error } = useSWR('/v1/identities')
  const { admin, loading: adminLoading } = useAdmin()
  const table = useTable({ columns, data: users || [] })
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [slideModalOpen, setSlideModalOpen] = useState(false)
  const [selectedRow, setSelectedRow] = useState(null)
  const [slideActionBtns, setSlideActionBtns] = useState([])

  const loading = adminLoading || (!users && !error)

  const handleUserDetail = (row) => {
    setSlideModalOpen(true)
    setSelectedRow(row)
    setSlideActionBtns([{ handleOnClick: () => setDeleteModalOpen(true), text: 'Remove User' }])
  }

  const handleCancelDeleteModal = () => {
    setDeleteModalOpen(false)
    setSlideModalOpen(true)
  }

  return (
    <>
      <Head>
        <title>Users - Infra</title>
      </Head>
      {loading
        ? (<Loader />)
        : (
          <div className={`flex-1 flex flex-col space-y-8 mt-3 mb-4 ${slideModalOpen ? 'w-7/12' : ''} h-screen`}>
            <PageHeader header='Users' buttonHref={admin && '/users/add'} buttonLabel='User' />
            <div className='overflow-y-auto max-h-full'>
              {error?.status
                ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
                : <>
                  <Table
                    {...table}
                    getRowProps={row => ({
                      onClick: () => handleUserDetail(row),
                      style: {
                        cursor: 'pointer'
                      }
                    })}
                  />
                </>}
              {slideModalOpen &&
                <Slide
                  open={slideModalOpen}
                  handleClose={() => setSlideModalOpen(false)}
                  title={selectedRow.original.name}
                  footerBtns={slideActionBtns}
                  profileIconName={selectedRow.original.name[0]}
                  deleteModalShown={deleteModalOpen}
                >
                  <SlideContent id={selectedRow.original.id} isAdmin={admin} />
                </Slide>}
              <DeleteModal
                open={deleteModalOpen}
                setOpen={setDeleteModalOpen}
                onCancel={handleCancelDeleteModal}
                onSubmit={async () => {
                  mutate('/v1/identities', async users => {
                    await fetch(`/v1/identities/${selectedRow.original.id}`, {
                      method: 'DELETE'
                    })

                    return users?.filter(u => u?.id !== selectedRow.original.id)
                  })

                  setDeleteModalOpen(false)
                }}
                title='Delete User'
                message={<>Are you sure you want to remove <span className='text-white font-bold'>{selectedRow?.original.name}?</span></>}
              />
              {
                users?.length === 0 &&
                  <EmptyTable
                    title='There are no users'
                    subtitle='Invite users to Infra and manage their access.'
                    iconPath='/users.svg'
                    buttonHref={admin && '/users/add'}
                    buttonText='Users'
                  />
              }
            </div>
          </div>
          )}
    </>
  )
}

Users.layout = function (page) {
  return (
    <Dashboard>{page}</Dashboard>
  )
}
