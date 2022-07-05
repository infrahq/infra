import Head from 'next/head'
import { useRouter } from 'next/router'
import useSWR from 'swr'
import { useState } from 'react'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import PageHeader from '../../components/page-header'
import EmptyTable from '../../components/empty-table'
import Table from '../../components/table'
import Sidebar from '../../components/sidebar'
import DeleteModal from '../../components/delete-modal'

const columns = [
  {
    Header: 'Name',
    accessor: g => g,
    width: '80%',
    Cell: ({ value: group }) => {
      return (
        <div className='flex items-center py-2'>
          <div className='flex h-6 w-6 select-none items-center justify-center rounded-md border border-violet-300/40'>
            <img alt='group icon' src='/groups.svg' className='h-3 w-3' />
          </div>
          <div className='ml-3 flex min-w-0 flex-1 flex-col leading-tight'>
            <div className='truncate'>{group.name}</div>
          </div>
        </div>
      )
    },
  },
  {
    Header: 'Team Size',
    accessor: g => g,
    width: '20%',
    Cell: ({ value: group }) => {
      const { data: { items: users } = {}, error } = useSWR(
        `/api/users?group=${group.id}`
      )

      return (
        <>
          {users && (
            <div className='text-gray-400'>
              {users?.length} {users?.length > 1 ? 'members' : 'member'}
            </div>
          )}
          {error?.status && <div className='text-gray-400'>--</div>}
        </>
      )
    },
  },
]

function Details({ group, admin }) {
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)

  return (
    <div className='flex flex-1 flex-col space-y-6'>
      {admin && (
        <>
          <section>
            <h3 className='mb-4 border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
              Access
            </h3>
          </section>
          <section>
            <h3 className='mb-4 border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
              Team
            </h3>
          </section>
        </>
      )}
      <section>
        <h3 className='mb-4 border-b border-gray-800 py-4 text-3xs uppercase text-gray-400'>
          Metadata
        </h3>
      </section>
      {admin && (
        <section className='flex flex-1 flex-col items-end justify-end py-6'>
          <button
            type='button'
            onClick={() => setDeleteModalOpen(true)}
            className='flex items-center rounded-md border border-violet-300 px-6 py-3 text-2xs text-violet-100'
          >
            Remove
          </button>
          <DeleteModal
            open={deleteModalOpen}
            setOpen={setDeleteModalOpen}
            onSubmit={() => {}}
            title='Remove Group'
            message={
              <>
                Are you sure you want to delete{' '}
                <span className='font-bold text-white'>{group?.name}</span>?
                This action cannot be undone.
              </>
            }
          />
        </section>
      )}
    </div>
  )
}

export default function Groups() {
  const { data: { items: groups } = {}, error } = useSWR('/api/groups')
  const { admin, loading: adminLoading } = useAdmin()
  const router = useRouter()

  const loading = adminLoading || (!groups && !error)
  const { slug: [id] = [] } = router.query

  const group = groups?.find(g => g.id === id)

  if (id && groups && !group) {
    router.replace('/groups')
    return null
  }

  return (
    <>
      <Head>Groups - Infra</Head>
      {!loading && (
        <div className='flex h-full flex-1'>
          <div className='flex flex-1 flex-col space-y-4'>
            <PageHeader
              header='Groups'
              buttonHref='/groups/add'
              buttonLabel='Group'
            />
            {error?.status ? (
              <div className='my-20 text-center text-sm font-light text-gray-300'>
                {error?.info?.message}
              </div>
            ) : (
              <div className='flex min-h-0 flex-1 flex-col overflow-y-scroll px-6'>
                <Table
                  columns={columns}
                  data={
                    groups?.sort((a, b) =>
                      b.created?.localeCompare(a.created)
                    ) || []
                  }
                  getRowProps={row => ({
                    onClick: () => router.push(`/groups/${row.original.id}`),
                    className:
                      id === row.original.id
                        ? 'bg-gray-900/50'
                        : 'cursor-pointer',
                  })}
                />
                {groups?.length === 0 && (
                  <EmptyTable
                    title='There are no groups'
                    subtitle='Connect, create and manage your groups.'
                    iconPath='/groups.svg'
                    buttonHref='/groups/add'
                    buttonText='Groups'
                  />
                )}
              </div>
            )}
          </div>
          {id && (
            <Sidebar
              handleClose={() => router.push('/groups')}
              title={group?.name}
              iconPath='/groups.svg'
            >
              <Details group={groups?.find(g => g.id === id)} admin={admin} />
            </Sidebar>
          )}
        </div>
      )}
    </>
  )
}

Groups.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
