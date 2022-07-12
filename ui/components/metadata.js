export default function Metadata({ data }) {
  return (
    <>
      {data.map(item => (
        <div key={item.title} className='grid grid-cols-3 gap-x-1 gap-y-2 pt-3'>
          <div className='col-span-1 text-2xs text-gray-400'>{item.title}</div>
          <div className='col-span-2 text-2xs'>{item.data}</div>
        </div>
      ))}
    </>
  )
}
