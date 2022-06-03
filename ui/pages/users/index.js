import Head from 'next/head'
import { useState } from 'react'
import { useTable } from 'react-table'
import useSWR, { mutate } from 'swr'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'

import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/page-header'
import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'
import Sidebar from '../../components/sidebar'
import ProfileIcon from '../../components/profile-icon'
import DeleteModal from '../../components/modals/delete'
import ResourcesGrant from '../../components/resources-grant'

const columns = [{
  Header: 'Name',
  accessor: u => u,
  Cell: ({ value: user }) => (
    <div className='flex items-center py-1.5'>
      <ProfileIcon name={user.name[0]} />
      <div className='flex flex-col leading-tight ml-3'>
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
    <div className='text-name text-gray-400'>{user?.created ? dayjs(user.created).fromNow() : '-'}</div>
  )
}]

function SidebarContent ({ selectedUser, admin, setSelectedUser }) {
  const { id, name } = selectedUser
  const { data: user } = useSWR(`/api/users/${id}`)
  const { data: { items: grants } = {} } = useSWR(`/api/grants?user=${id}`)
  const { data: auth } = useSWR('/api/users/self')

  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  const infraRole = grants?.filter(g => g.resource === 'infra')?.[0]?.privilege

  return (
    <div className='flex-1 flex flex-col space-y-6'>
      {admin &&
        <section>
          <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Access</h3>
          <ResourcesGrant id={id} />
        </section>}
      <section>
        <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Metadata</h3>
        <div className='pt-3 flex flex-col space-y-2'>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>User</div>
            <div className='text-2xs'>{user?.name}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Infra Role</div>
            <div className='text-2xs'>{infraRole || '-'}</div>
          </div>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>Created</div>
            <div className='text-2xs'>{user?.created ? dayjs(user.created).fromNow() : '-'}</div>
          </div>
        </div>
      </section>
      <section className='flex-1 flex flex-col items-end justify-end py-6'>
        {auth.id !== id &&
          <button
            type='button'
            onClick={() => setDeleteModalOpen(true)}
            className='border border-violet-300 rounded-md flex items-center text-2xs px-6 py-3 text-violet-100'
          >
            Remove
          </button>}
        <DeleteModal
          open={deleteModalOpen}
          setOpen={setDeleteModalOpen}
          onSubmit={async () => {
            mutate('/api/users', async ({ items: users } = { items: [] }) => {
              await fetch(`/api/users/${id}`, {
                method: 'DELETE'
              })

              return { items: users?.filter(u => u?.id !== id) }
            })

            setDeleteModalOpen(false)
            setSelectedUser(null)
          }}
          title='Remove User'
          message={<>Are you sure you want to remove <span className='text-white font-bold'>{name}?</span></>}
        />
      </section>
    </div>
  )
}

export default function Users () {
  const { data: { items } = {}, error } = useSWR('/api/users')
  const { admin, loading: adminLoading } = useAdmin()
  const users = items?.filter(u => u.name !== 'connector')
  const table = useTable({ columns, data: users?.sort((a, b) => b.created.localeCompare(a.created)) || [] })
  const [selectedUser, setSelectedUser] = useState(null)

  const loading = adminLoading || (!users && !error)

  return (
    <>
      <Head>
        <title>Users - Infra</title>
      </Head>
      {!loading && (
        <div className='flex-1 flex h-full'>
          <main className='flex-1 flex flex-col space-y-4'>
            <PageHeader header='Users' buttonHref={admin && '/users/add'} buttonLabel='User' />
            {error?.status
              ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
              : (
                <div>
                  <Table
                    {...table}
                    getRowProps={row => ({
                      onClick: () => setSelectedUser(row.original),
                      style: {
                        cursor: 'pointer'
                      }
                    })}
                  />
                  {users?.length === 0 && <EmptyTable
                    title='There are no users'
                    subtitle='Invite users to Infra and manage their access.'
                    iconPath='/users.svg'
                    buttonHref={admin && '/users/add'}
                    buttonText='Users'
                                          />}
                </div>
                )}
          </main>
          {selectedUser &&
            <Sidebar
              handleClose={() => setSelectedUser(null)}
              title={selectedUser.name}
              profileIcon={selectedUser.name[0]}
            >
              <SidebarContent selectedUser={selectedUser} admin={admin} setSelectedUser={setSelectedUser} />
            </Sidebar>}
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
