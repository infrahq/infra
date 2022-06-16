export default function (props) {
  return (
    <iframe
      {...props}
      frameBorder='0'
      allow='accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture'
      allowFullScreen
      className='aspect-video w-full my-10'
    />
  )
}
