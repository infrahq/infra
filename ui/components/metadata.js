export default function Metadata({ data }) {
  return (
    <div className='flex flex-col space-y-2 pt-3'>
      {data.map(item => (
        <div key={item.title} className='flex flex-row items-center'>
          <div className='w-1/3 text-2xs text-gray-400'>{item.title}</div>
          <div className='text-2xs'>{item.data}</div>
        </div>
      ))}
    </div>
  )
}
