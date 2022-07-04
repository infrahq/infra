import Head from 'next/head'
import useSWR from 'swr'
import { useTable } from 'react-table'
import { useState } from 'react'

import Dashboard from '../../components/layouts/dashboard'
import PageHeader from '../../components/page-header'
import EmptyTable from '../../components/empty-table'
import Table from '../../components/table'

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

export default function Groups() {
  const { data: { items: groups } = {}, error } = useSWR('/api/groups')
  const table = useTable({ columns, data: groups || [] })

  console.log(groups)

  const [selected, setSelected] = useState(null)

  const loading = !groups && !error

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
                  {...table}
                  getRowProps={row => ({
                    onClick: () => setSelected(row.original),
                    className:
                      selected?.id === row.original.id
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
        </div>
      )}
    </>
  )
}

Groups.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
