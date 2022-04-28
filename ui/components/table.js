import { useTable } from 'react-table'

export default function ({ columns, data, getRowProps = () => ({}), showHeader = true }) {
  const { getTableProps, getTableBodyProps, headerGroups, rows, prepareRow } = useTable({
    columns,
    data
  })

  return (
    <table className='w-full table-auto' {...getTableProps()}>
      {showHeader &&
        <thead className='border-b border-zinc-800'>
          {headerGroups.map(headerGroup => (
            <tr key={headerGroup.id} {...headerGroup.getHeaderGroupProps()}>
              {headerGroup.headers.map(column => (
                <th key={column.id} className='text-left uppercase py-2 font-normal text-xs text-secondary' {...column.getHeaderProps()}>
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
            <tr className='text-xs group' key={row.id} {...row.getRowProps(getRowProps(row))}>
              {row.cells.map(cell => {
                return (
                  <td key={cell.id} className='py-1 group-first:pt-3' {...cell.getCellProps()}>
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
