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
    Cell: ({ value: group }) => <div>{group.name}</div>,
  },
  {
    Header: 'Team Size',
    accessor: g => g,
    width: '25%',
    Cell: ({ value: group }) => {
      const { data: { items: users } = {}, error } = useSWR(
        `/api/users?group=${group.id}`
      )
      const loading = !users && !error

      console.log(users)

      return <>{/* {!loading && (<div>{users.length}</div>)} */}</>
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
