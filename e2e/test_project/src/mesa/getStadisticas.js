const { Client } = require('pg')
const { validator:{validateAll} } = require('indicative') 
const { getRun } = require('util-header')

module.exports.handler = async (event, context) => {
    
    context.callbackWaitsForEmptyEventLoop = false
    let run = null
    try {
        run = await getRun(event) 
        
    } catch (error) {
        console.error('Error al obtener el run del header:: ', error)
        return { statusCode: 200 , ok : false }
    }

    const client = new Client() 

    
    try {
        await client.connect()

         const { rows } =  await client.query(
            ` 
            SELECT 
            to_char(fecha_digitacion, 'YYYY-MM-DD') as fecha,
              CASE estado
                WHEN 1 THEN 'digitado_ok'
                WHEN 2 THEN 'requiere_validacion'
                WHEN 3 THEN 'problemas_doc'
                WHEN 4 THEN 'mesa_descuadrada'
              END AS nombre_estado,
              COUNT(*) AS total_mesas,
              SUM(CASE WHEN usuario_id = 1 THEN 1 ELSE 0 END) AS "Christian fajardo",
              SUM(CASE WHEN usuario_id = 2 THEN 1 ELSE 0 END) AS "Pablo Guajardo",
              SUM(CASE WHEN usuario_id = 3 THEN 1 ELSE 0 END) AS "Jose Riquelme",
              SUM(CASE WHEN usuario_id = 4 THEN 1 ELSE 0 END) AS "Jean Guerrero",
              SUM(CASE WHEN usuario_id = 5 THEN 1 ELSE 0 END) AS "Samuel Morales",
              SUM(CASE WHEN usuario_id = 6 THEN 1 ELSE 0 END) AS "Leopoldo Soto",
              SUM(CASE WHEN usuario_id = 7 THEN 1 ELSE 0 END) AS "Meliodas Gremory",
              SUM(CASE WHEN usuario_id = 8 THEN 1 ELSE 0 END) AS "Kevin Orellana",
              SUM(CASE WHEN usuario_id = 9 THEN 1 ELSE 0 END) AS "Jose Peraldi",
              SUM(CASE WHEN usuario_id = 10 THEN 1 ELSE 0 END) AS "Andres Torres"
            FROM mesa
            WHERE estado IS NOT NULL
            GROUP BY estado, fecha
            order by fecha desc
            `)    

            const totales =  await client.query(
                ` 
                SELECT 
                    COUNT(*) FILTER (WHERE estado IS NULL and "REGIONID" <>'13') AS por_digitar,
                    COUNT(*) FILTER (WHERE estado IS NOT NULL and "REGIONID" <>'13') AS digitados,
                    COUNT(*) FILTER (WHERE estado = 1 and "REGIONID" <>'13') AS digitados_ok,
                    COUNT(*) FILTER (WHERE (estado = 2 or estado = 4) and "REGIONID" <>'13') AS digitado_con_inconsistencia,
                    COUNT(*) FILTER (WHERE (estado = 3 or estado =6 or estado= 5) and "REGIONID" <>'13') AS problema_doc,
                        COUNT(*) FILTER (WHERE estado = 7) AS digitacion_corregida
                FROM 
                    mesa;
                `) 



        await client.end()

        return { statusCode: 200 , data : {totales:totales.rows, estadisticas:rows } }

    }catch(error){
        console.error('Fallo getStadisticas::', error)
        if(client)
        {
            await client.end()

            return { statusCode: 200 , ok : false }
        }


    }
}