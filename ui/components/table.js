import { useTable, useExpanded } from 'react-table'

export default function Table({ columns, data, getRowProps = () => {} }) {
  const { getTableProps, getTableBodyProps, prepareRow, headerGroups, rows } =
    useTable(
      {
        columns,
        data,
        autoResetExpanded: false,
      },
      useExpanded
    )

  return (
    <table
      className='w-full table-fixed border-separate border-spacing-0'
      {...getTableProps()}
    >
      <thead>
        {headerGroups.map(headerGroup => (
          <tr key={headerGroup.id} {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map(column => (
              <th
                width={column.width}
                key={column.id}
                className='sticky top-0 z-10 border-b border-gray-800 bg-black py-1 text-left text-3xs font-normal uppercase text-gray-400'
                {...column.getHeaderProps()}
              >
                {column.render('Header')}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody className='relative' {...getTableBodyProps()}>
        {rows.map(row => {
          prepareRow(row)
          const props = row.getRowProps(getRowProps(row))
          return (
            <tr
              {...props}
              key={row.id}
              className={`${props.className} group text-2xs`}
            >
              {row.cells.map(cell => {
                const props = cell.getCellProps()
                return (
                  <td
                    key={cell.id}
                    {...props}
                    className={`${
                      props.className || ''
                    } border-b border-gray-800`}
                  >
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
