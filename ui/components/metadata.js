export default function Metadata({ data }) {
  return (
    <>
      {data.map(item => (
        <div
          key={item.title}
          data-testid='metadata-item'
          className='grid grid-cols-3 gap-x-1 gap-y-2 pt-3'
        >
          <div
            data-testid='metadata-title'
            className='col-span-1 text-2xs text-gray-400'
          >
            {item.title}
          </div>
          <div
            data-testid='metadata-data'
            className='col-span-2 break-words text-2xs'
          >
            {item.data}
          </div>
        </div>
      ))}
    </>
  )
}
