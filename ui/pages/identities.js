import useSWR from 'swr'
import { useTable } from 'react-table'
import dayjs from 'dayjs'

import Dashboard from '../components/dashboard'

const columns = [
  {
    Header: 'Identity',
    accessor: 'name',
    Cell: ({ value }) => (
      <div className='flex items-center'>
        <div className='w-9 h-9 mr-4 bg-purple-100/10 font-bold rounded-lg flex items-center justify-center'>{value[0]?.toUpperCase()}</div>
        <div>{value}</div>
      </div>
    )
  },
  {
    accessor: 'kind', // accessor is the "key" in the data,
    Header: () => (
      <div className='text-right'>
        Kind
      </div>
    ),
    Cell: ({ value }) => (
      <div className='text-right'>
        {value}
      </div>
    )
  },
  {
    id: 'last_seen',
    accessor: i => {
      return dayjs(i.lastSeenAt).fromNow()
    },
    Header: () => (
      <div className='text-right'>
        Last Seen
      </div>
    ),
    Cell: ({ value }) => (
      <div className='text-right'>
        {value}
      </div>
    )
  }
]

export default function () {
  const { data, error } = useSWR('/v1/identities', { fallbackData: [] })

  const identities = error ? [] : data
  console.log(identities, error)
  const table = useTable({ columns, data: identities })

  const {
    getTableProps,
    getTableBodyProps,
    headerGroups,
    rows,
    prepareRow
  } = table

  return (
    <Dashboard>
      <div className='my-20'>
        <h1 className='text-4xl font-bold my-8'>Identities</h1>
        <table className='w-full table-auto' {...getTableProps()}>
          <thead className='border-b border-gray-900'>
            {headerGroups.map(headerGroup => (
              <tr key={headerGroup.id} {...headerGroup.getHeaderGroupProps()}>
                {headerGroup.headers.map(column => (
                  <th key={column.id} className='text-left py-1 font-normal text-sm text-gray-400' {...column.getHeaderProps()}>
                    {column.render('Header')}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody {...getTableBodyProps()}>
            {rows.map(row => {
              prepareRow(row)
              return (
                <tr key={row.id} {...row.getRowProps()}>
                  {row.cells.map(cell => {
                    return (
                      <td key={cell.id} className='py-1.5' {...cell.getCellProps()}>
                        {cell.render('Cell')}
                      </td>
                    )
                  })}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </Dashboard>
  )
}
