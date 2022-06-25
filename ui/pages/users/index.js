import Head from 'next/head'
import { useState } from 'react'
import { useTable } from 'react-table'
import useSWR from 'swr'
import dayjs from 'dayjs'

import { useAdmin } from '../../lib/admin'
import { editGrant, removeGrant, sortByResource } from '../../lib/grants'
import EmptyTable from '../../components/empty-table'
import PageHeader from '../../components/page-header'
import Table from '../../components/table'
import Dashboard from '../../components/layouts/dashboard'
import Sidebar from '../../components/sidebar'
import ProfileIcon from '../../components/profile-icon'
import DeleteModal from '../../components/modals/delete'
import RoleDropdown from '../../components/role-dropdown'

const columns = [{
  Header: 'Name',
  accessor: u => u,
  Cell: ({ value: user }) => (
    <div className='flex items-center py-1.5'>
      <ProfileIcon name={user.name[0]} />
      <div className='flex-1 min-w-0 flex flex-col leading-tight ml-3'>
        <div className='truncate'>{user.name}</div>
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

function Details ({ user, admin, onDelete }) {
  const { id, name } = user
  const { data: auth } = useSWR('/api/users/self')

  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  const { data: { items } = {}, mutate } = useSWR(`/api/grants?user=${id}`)
  const { data: { items: groups } = {} } = useSWR(`/api/groups?userID=${id}`)
  const { data: groupGrantDatas } = useSWR(
    () => groups ? groups.map(g => `/api/grants?group=${g.id}`) : null,
    (...urls) => Promise.all(urls.map(url => fetch(url).then(r => r.json())))
  )

  const grants = items?.filter(g => g.resource !== 'infra')
  const inherited = groupGrantDatas?.map(g => g?.items || [])?.flat()?.filter(g => g.resource !== 'infra')

  const loading = [auth, grants, groups, groups?.length ? inherited : true].some(x => !x)

  return (
    <div className='flex-1 flex flex-col space-y-6'>
      {admin &&
        <section>
          <h3 className='py-4 mb-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Access</h3>
          {grants?.sort(sortByResource)?.map(g => (
            <div key={g.id} className='flex justify-between items-center text-2xs'>
              <div>{g.resource}</div>
              <RoleDropdown
                role={g.privilege}
                resource={g.resource}
                remove
                direction='left'
                onRemove={() => mutate(removeGrant(g.id))}
                onChange={privilege => mutate(editGrant(g.id, { ...g, privilege }))}
              />
            </div>
          ))}
          {inherited?.sort(sortByResource)?.map(g => (
            <div key={g.id} className='flex justify-between items-center text-2xs'>
              <div>{g.resource}</div>
              <div className='flex-none flex'>
                <div
                  title='This access is inherited and cannot be edited here'
                  className='relative pt-px mx-1 self-center text-2xs text-gray-400 border rounded px-2 bg-gray-800 border-gray-800'
                >
                  inherited
                </div>
                <div className='relative flex-none pl-3 pr-8 w-32 py-2 text-left text-2xs text-gray-400'>
                  {g.privilege}
                </div>
              </div>
            </div>
          ))}
          {!grants?.length && !inherited?.length && !loading && (
            <div className='text-2xs text-gray-400 mt-6 italic'>No access</div>
          )}
        </section>}
      <section>
        <h3 className='py-4 text-3xs text-gray-400 border-b border-gray-800 uppercase'>Metadata</h3>
        <div className='pt-3 flex flex-col space-y-2'>
          <div className='flex flex-row items-center'>
            <div className='text-gray-400 text-2xs w-1/3'>ID</div>
            <div className='text-2xs'>{user?.id}</div>
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
            setDeleteModalOpen(false)
            onDelete()
          }}
          title='Remove User'
          message={<>Are you sure you want to remove <span className='text-white font-bold'>{name}?</span></>}
        />
      </section>
    </div>
  )
}

export default function Users () {
  const { data: { items } = {}, error, mutate } = useSWR('/api/users')
  const { admin, loading: adminLoading } = useAdmin()
  const users = items?.filter(u => u.name !== 'connector')
  const table = useTable({ columns, data: users?.sort((a, b) => b.created?.localeCompare(a.created)) || [] })
  const [selected, setSelected] = useState(null)

  const loading = adminLoading || (!users && !error)

  return (
    <>
      <Head>
        <title>Users - Infra</title>
      </Head>
      {!loading && (
        <div className='flex-1 flex h-full'>
          <div className='flex-1 flex flex-col h-full'>
            <PageHeader header='Users' buttonHref={admin && '/users/add'} buttonLabel='User' />
            {error?.status
              ? <div className='my-20 text-center font-light text-gray-300 text-sm'>{error?.info?.message}</div>
              : (
                <div className='flex flex-col flex-1 px-6 min-h-0 overflow-y-scroll'>
                  <Table
                    {...table}
                    getRowProps={row => ({
                      onClick: () => setSelected(row.original),
                      className: selected?.id === row.original.id ? 'bg-gray-900/50' : 'cursor-pointer'
                    })}
                  />
                  {users?.length === 0 &&
                    <EmptyTable
                      title='There are no users'
                      subtitle='Invite users to Infra and manage their access.'
                      iconPath='/users.svg'
                      buttonHref={admin && '/users/add'}
                      buttonText='Users'
                    />}
                </div>
                )}
          </div>
          {selected &&
            <Sidebar handleClose={() => setSelected(null)} title={selected.name} profileIcon={selected.name[0]}>
              <Details
                user={selected}
                admin={admin}
                onDelete={() => {
                  mutate(async ({ items: users } = { items: [] }) => {
                    await fetch(`/api/users/${selected.id}`, {
                      method: 'DELETE'
                    })

                    return { items: users?.filter(u => u?.id !== selected.id) }
                  })

                  setSelected(null)
                }}
              />
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
