import { useTable } from 'react-table'

export default function ({ columns, data, getRowProps = () => ({}), showHeader = true }) {
  const { getTableProps, getTableBodyProps, headerGroups, rows, prepareRow } = useTable({
    columns,
    data
  })

  return (
    <table className='w-full table-auto' {...getTableProps()}>
      {showHeader &&
        <thead className='border-b border-gray-800'>
          {headerGroups.map(headerGroup => (
            <tr key={headerGroup.id} {...headerGroup.getHeaderGroupProps()}>
              {headerGroup.headers.map(column => (
                <th key={column.id} className='text-left uppercase px-2.5 py-4 font-normal text-label text-gray-400' {...column.getHeaderProps()}>
                  {column.render('Header')}
                </th>
              ))}
            </tr>
          ))}
        </thead>}
      <tbody {...getTableBodyProps()}>
        {rows.map(row => {
          prepareRow(row)
          return (
            <tr className='table-flex text-sm group border-b border-gray-800 hover:bg-gray-350/50 shadow hover:shadow-lg' key={row.id} {...row.getRowProps(getRowProps(row))}>
              {row.cells.map(cell => {
                return (
                  <td key={cell.id} className={`px-2.5 py-2 ${showHeader ? 'group-first:pt-3' : ''}`} {...cell.getCellProps()}>
                    {cell.render('Cell')}
                  </td>
                )
              })}
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}
